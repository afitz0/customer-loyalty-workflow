package starter

import (
	"fmt"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/temporal"
	"time"

	"go.temporal.io/sdk/workflow"
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
			fmt.Sprintf(EmailWelcome, StatusTiers[customer.StatusLevel].Name)).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity.", "Error", err)
			return err
		}
	}

	// signal handler for adding points
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalAddPoints),
		func(c workflow.ReceiveChannel, _ bool) {
			errSignal = signalAddPoints(ctx, c, &customer)
		})

	// signal handler for adding guest
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalInviteGuest),
		func(c workflow.ReceiveChannel, _ bool) {
			errSignal = signalInviteGuest(ctx, c, &customer)
		})

	// signal handler for ensuring the customer is at least the given status. Used for invites and promoting an existing account.
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalEnsureMinimumStatus),
		func(c workflow.ReceiveChannel, _ bool) {
			errSignal = signalEnsureMinimumStatus(ctx, c, &customer)
		})

	// signal handler for canceling account
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalCancelAccount),
		func(c workflow.ReceiveChannel, _ bool) {
			errSignal = signalCancelAccount(ctx, c, &customer)
		})

	// query handler for status level, etc
	err = workflow.SetQueryHandler(ctx, QueryGetStatus,
		func() (GetStatusResponse, error) {
			return queryGetStatus(ctx, customer)
		})

	// query handler for guest list
	err = workflow.SetQueryHandler(ctx, QueryGetGuests,
		func() ([]string, error) {
			return queryGetGuests(ctx, customer)
		})

	// Block on everything. Continue-As-New on history length; size of activities in this workflow are small enough
	// that we'll hit the length thresholds well before any size threshold.
	for customer.AccountActive && info.GetCurrentHistoryLength() < EventsThreshold {
		selector.Select(ctx)

		if errSignal != nil {
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
		WorkflowID:            fmt.Sprintf(CustomerWorkflowIdFormat, guest.CustomerId),
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
	if customer.StatusLevel >= len(StatusTiers) {
		customer.StatusLevel = len(StatusTiers) - 1
	}
	if customer.StatusLevel < 0 {
		customer.StatusLevel = 0
	}

	if len(customer.Guests) == 0 {
		customer.Guests = make(map[string]struct{}, StatusTiers[customer.StatusLevel].GuestsAllowed)
	}
}

func signalAddPoints(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) (err error) {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

	var pointsToAdd int
	c.Receive(ctx, &pointsToAdd)

	logger.Info("Adding points to customer account.", "PointsAdded", pointsToAdd)
	customer.LoyaltyPoints += pointsToAdd

	promoted := false
	// while customer's current points are higher than next status, increase their status
	for customer.LoyaltyPoints >= StatusTiers[min(len(StatusTiers)-1, customer.StatusLevel+1)].MinimumPoints &&
		customer.StatusLevel < len(StatusTiers)-1 {
		customer.StatusLevel++
		promoted = true
	}

	if promoted {
		err = workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(EmailPromoted, StatusTiers[customer.StatusLevel].Name)).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity.", "Error", err)
			return err
		}
	}

	return nil
}

func signalInviteGuest(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) (err error) {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

	var emailToSend string

	if len(customer.Guests) < StatusTiers[customer.StatusLevel].GuestsAllowed {
		var guestId string
		c.Receive(ctx, &guestId)

		logger.Info("Customer is allowed to invite guests. Attempting to invite.",
			"GuestId", guestId)

		guest := CustomerInfo{
			CustomerId:    guestId,
			AccountActive: true,
		}

		customer.Guests[guestId] = struct{}{}
		previousTier := StatusTiers[max(customer.StatusLevel-1, 0)]

		err, guestWorkflow := startGuestWorkflow(ctx, guest)
		logger.Info("Results from starting guest.", "error", err, "guest", guestWorkflow)

		if _, ok := err.(*serviceerror.WorkflowExecutionAlreadyStarted); ok {
			emailToSend = EmailGuestInvited
			err = guestWorkflow.SignalChildWorkflow(ctx, SignalEnsureMinimumStatus, previousTier).
				Get(ctx, nil)
			if _, ok := err.(*serviceerror.WorkflowExecutionAlreadyStarted); ok {
				logger.Info("Failed to signal 'already started' guest account; child workflow likely closed.")
				emailToSend = EmailGuestCanceled
			} else if err != nil {
				logger.Error("Could not signal guest/child workflow.")
				return err
			}
		} else if err != nil {
			logger.Error("Could not start guest/child workflow.")
			return err
		}
	} else {
		logger.Info("Customer does not have sufficient status to invite guests.")
		emailToSend = EmailInsufficientPoints
	}

	err = workflow.ExecuteActivity(ctx, activities.SendEmail, emailToSend).Get(ctx, nil)
	if err != nil {
		logger.Error("Error running SendEmail activity.", "Error", err)
		return err
	}

	return nil
}

func signalEnsureMinimumStatus(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) (err error) {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

	var minStatus StatusTier
	c.Receive(ctx, &minStatus)

	promoted := false
	for StatusTiers[customer.StatusLevel].MinimumPoints < minStatus.MinimumPoints && customer.StatusLevel < len(StatusTiers)-1 {
		customer.StatusLevel++
		promoted = true
	}

	if promoted {
		emailBody := fmt.Sprintf(EmailPromoted, minStatus.Name)
		err := workflow.ExecuteActivity(ctx, activities.SendEmail, emailBody).Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity.", "Error", err)
			return err
		}
	}

	return nil
}

func signalCancelAccount(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) (err error) {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

	// nothing to receive, but need this to "handle" signal
	c.Receive(ctx, nil)

	customer.AccountActive = false
	err = workflow.ExecuteActivity(ctx, activities.SendEmail, EmailCancelAccount).Get(ctx, nil)
	if err != nil {
		logger.Error("Error running SendEmail activity.", "Error", err)
		return err
	}

	logger.Info("Canceled account.", "CustomerID", customer.CustomerId)
	return nil
}

func queryGetStatus(ctx workflow.Context, customer CustomerInfo) (GetStatusResponse, error) {
	logger := workflow.GetLogger(ctx)

	status := GetStatusResponse{
		StatusLevel:   customer.StatusLevel,
		Tier:          StatusTiers[customer.StatusLevel],
		Points:        customer.LoyaltyPoints,
		AccountActive: customer.AccountActive,
	}
	logger.Info("Got status query.", "Customer", customer, "Response", status)

	return status, nil
}

func queryGetGuests(ctx workflow.Context, customer CustomerInfo) ([]string, error) {
	logger := workflow.GetLogger(ctx)
	guestIds := make([]string, len(customer.Guests))

	i := 0
	for k := range customer.Guests {
		guestIds[i] = k
		i++
	}

	logger.Info("Got guest list query.", "Guests", guestIds)
	return guestIds, nil
}
