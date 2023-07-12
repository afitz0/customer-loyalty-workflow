package loyalty

import (
	"fmt"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/afitz0/customer-loyalty-workflow/common"
	"github.com/afitz0/customer-loyalty-workflow/status"
)

func CustomerLoyaltyWorkflow(ctx workflow.Context, customer CustomerInfo) (err error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Loyalty workflow started.", "CustomerInfo", customer)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: 60 * time.Second,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	info := workflow.GetInfo(ctx)
	selector := workflow.NewSelector(ctx)
	activities := Activities{}
	var errSignal error

	logger.Info("Validating customer info.")
	validateCustomerInfo(&customer)

	if info.ContinuedExecutionRunID == "" {
		err = workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(common.EmailWelcome, customer.Status.Name())).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity.", "Error", err)
			return err
		}
	}

	// signal handler for adding points
	selector.AddReceive(workflow.GetSignalChannel(ctx, common.SignalAddPoints),
		func(c workflow.ReceiveChannel, _ bool) {
			errSignal = signalAddPoints(ctx, c, &customer)
		})

	// signal handler for adding guest
	selector.AddReceive(workflow.GetSignalChannel(ctx, common.SignalInviteGuest),
		func(c workflow.ReceiveChannel, _ bool) {
			errSignal = signalInviteGuest(ctx, c, &customer)
		})

	// signal handler for ensuring the customer is at least the given status. Used for invites and promoting an existing account.
	selector.AddReceive(workflow.GetSignalChannel(ctx, common.SignalEnsureMinimumStatus),
		func(c workflow.ReceiveChannel, _ bool) {
			errSignal = signalEnsureMinimumStatus(ctx, c, &customer)
		})

	// signal handler for canceling account
	selector.AddReceive(workflow.GetSignalChannel(ctx, common.SignalCancelAccount),
		func(c workflow.ReceiveChannel, _ bool) {
			errSignal = signalCancelAccount(ctx, c, &customer)
		})

	// query handler for status level, etc
	err = workflow.SetQueryHandler(ctx, common.QueryGetStatus,
		func() (GetStatusResponse, error) {
			return queryGetStatus(ctx, customer)
		})

	// query handler for guest list
	err = workflow.SetQueryHandler(ctx, common.QueryGetGuests,
		func() ([]string, error) {
			return queryGetGuests(ctx, customer)
		})

	// Block on everything. Continue-As-New on history length; size of activities in this workflow are small enough
	// that we'll hit the length thresholds well before any size threshold.
	for customer.AccountActive && info.GetCurrentHistoryLength() < common.EventsThreshold {
		selector.Select(ctx)

		if errSignal != nil {
			logger.Error("Unrecoverable error in handling a signal.", "Error", errSignal)
			return errSignal
		}
	}

	// here because of events threshold, but account still active? Continue-As-New
	if customer.AccountActive {
		return workflow.NewContinueAsNewError(ctx, customer)
	}

	logger.Info("Loyalty workflow completed.", "Customer", customer)
	return nil
}

func startGuestWorkflow(ctx workflow.Context, guest CustomerInfo) (err error, child workflow.ChildWorkflowFuture) {
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		WorkflowID:            fmt.Sprintf(common.CustomerWorkflowIDFormat, guest.CustomerID),
		ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}
	ctx = workflow.WithChildOptions(ctx, childWorkflowOptions)

	child = workflow.ExecuteChildWorkflow(ctx, CustomerLoyaltyWorkflow, guest)
	// wait for it to start before returning.
	err = child.GetChildWorkflowExecution().Get(ctx, nil)
	return err, child
}

func validateCustomerInfo(customer *CustomerInfo) {
	customer.Status = status.NewStatus(customer.StatusLevel)
	if len(customer.Guests) == 0 {
		customer.Guests = make(map[string]struct{}, customer.Status.NumGuestsAllowed())
	}
}

func signalAddPoints(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) error {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

	var pointsToAdd int
	c.Receive(ctx, &pointsToAdd)

	logger.Info("Adding points to customer account.", "PointsAdded", pointsToAdd)
	customer.LoyaltyPoints += pointsToAdd

	statusChange := customer.Status.Update(customer.LoyaltyPoints)

	if statusChange > 0 {
		err := workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(common.EmailPromoted, customer.Status.Name())).
			Get(ctx, nil)
		if err != nil {
			return errors.Wrap(err, "Error running SendEmail activity for status promotion.")
		}
	} else if statusChange < 0 {
		err := workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(common.EmailDemoted, customer.Status.Name())).
			Get(ctx, nil)
		if err != nil {
			return errors.Wrap(err, "Error running SendEmail activity for status demotion.")
		}
	}

	return nil
}

func signalInviteGuest(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) error {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

	var emailToSend string

	if len(customer.Guests) < customer.Status.NumGuestsAllowed() {
		var guestID string
		c.Receive(ctx, &guestID)

		logger.Info("Customer is allowed to invite guests. Attempting to invite.",
			"GuestID", guestID)

		guest := CustomerInfo{
			CustomerID:    guestID,
			AccountActive: true,
		}

		customer.Guests[guestID] = struct{}{}

		err, guestWorkflow := startGuestWorkflow(ctx, guest)
		logger.Info("Results from starting guest.", "error", err, "guest", guestWorkflow)

		if _, ok := err.(*serviceerror.WorkflowExecutionAlreadyStarted); ok {
			emailToSend = common.EmailGuestInvited
			previousTier := customer.Status.PreviousTier()
			err = guestWorkflow.SignalChildWorkflow(ctx, common.SignalEnsureMinimumStatus, previousTier).
				Get(ctx, nil)
			if _, ok := err.(*serviceerror.WorkflowExecutionAlreadyStarted); ok {
				logger.Info("Failed to signal 'already started' guest account; child workflow likely closed.")
				emailToSend = common.EmailGuestCanceled
			} else if err != nil {
				return errors.Wrapf(err, "Could not signal guest/child workflow for guest ID '%v'.", guestID)
			}
		} else if err != nil {
			return errors.Wrapf(err, "Could not start guest/child workflow for guest ID '%v'.", guestID)
		}
	} else {
		logger.Info("Customer does not have sufficient status to invite more guests.")
		emailToSend = common.EmailInsufficientPoints
	}

	err := workflow.ExecuteActivity(ctx, activities.SendEmail, emailToSend).Get(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "Error running SendEmail activity.")
	}

	return nil
}

func signalEnsureMinimumStatus(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) error {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

	var minStatus status.Tier
	c.Receive(ctx, &minStatus)

	promoted := customer.Status.EnsureMinimum(minStatus)

	if promoted {
		emailBody := fmt.Sprintf(common.EmailPromoted, minStatus.Name)
		err := workflow.ExecuteActivity(ctx, activities.SendEmail, emailBody).Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity.", "Error", err)
			return err
		}
	}

	return nil
}

func signalCancelAccount(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) error {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

	// nothing to receive, but need this to "handle" signal
	c.Receive(ctx, nil)

	customer.AccountActive = false
	err := workflow.ExecuteActivity(ctx, activities.SendEmail, common.EmailCancelAccount).Get(ctx, nil)
	if err != nil {
		logger.Error("Error running SendEmail activity.", "Error", err)
		return err
	}

	logger.Info("Canceled account.", "CustomerID", customer.CustomerID)
	return nil
}

func queryGetStatus(ctx workflow.Context, customer CustomerInfo) (GetStatusResponse, error) {
	logger := workflow.GetLogger(ctx)

	response := GetStatusResponse{
		Tier:          customer.Status.Tier(),
		Points:        customer.LoyaltyPoints,
		AccountActive: customer.AccountActive,
	}
	logger.Info("Got response query.", "Customer", customer, "Response", response)

	return response, nil
}

func queryGetGuests(ctx workflow.Context, customer CustomerInfo) ([]string, error) {
	logger := workflow.GetLogger(ctx)
	guestIDs := make([]string, len(customer.Guests))

	i := 0
	for k := range customer.Guests {
		guestIDs[i] = k
		i++
	}

	logger.Info("Got guest list query.", "Guests", guestIDs)
	return guestIDs, nil
}
