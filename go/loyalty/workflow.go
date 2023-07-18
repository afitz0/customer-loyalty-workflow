package loyalty

import (
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const TaskQueue = "CustomerLoyaltyTaskQueue"
const EventsThreshold = 10000

// Signal, query, and error string constants
const (
	SignalCancelAccount       = "cancelAccount"
	SignalAddPoints           = "addLoyaltyPoints"
	SignalInviteGuest         = "inviteGuest"
	SignalEnsureMinimumStatus = "ensureMinimumStatus"
	QueryGetStatus            = "getStatus"
	QueryGetGuests            = "getGuests"
)

const (
	emailWelcome            = "Welcome to our loyalty program! You're starting out at '%v' status."
	emailGuestCanceled      = "Sorry, your guest has already canceled their account."
	emailGuestInvited       = "Congratulations! Your guest has been invited!"
	emailInsufficientPoints = "Sorry, you need to earn more points to invite more guests!"
	emailPromoted           = "Congratulations! You've been promoted to '%v' status!"
	emailDemoted            = "Unfortunately, you've lost enough points to bump you down to '%v' status. 😞"
	emailCancelAccount      = "Sorry to see you go!"
)

func CustomerLoyaltyWorkflow(ctx workflow.Context, customer CustomerInfo, newCustomer bool) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Loyalty workflow started.", "CustomerInfo", customer)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		// Slow retry with a hard limit. Used for sending emails.
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 10,
			InitialInterval: time.Second * 5,
		},
	}
	lao := workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			NonRetryableErrorTypes: []string{"GuestAlreadyCanceledError"},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	ctx = workflow.WithLocalActivityOptions(ctx, lao)

	info := workflow.GetInfo(ctx)
	selector := workflow.NewSelector(ctx)
	var activities Activities
	workflowCanceled := false
	var errSignal error

	logger.Info("Validating customer info.")
	customer.validate()

	if newCustomer {
		logger.Info("New customer workflow; sending welcome email.")
		err := workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(emailWelcome, customer.StatusLevel.Name)).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity for welcome email.", "Error", err)
		}
	} else {
		logger.Info("Continued workflow execution for existing customer.")
	}

	// signal handler for adding points
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalAddPoints),
		func(c workflow.ReceiveChannel, _ bool) {
			signalAddPoints(ctx, c, &customer)
		})

	// signal handler for adding guest
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalInviteGuest),
		func(c workflow.ReceiveChannel, _ bool) {
			errSignal = signalInviteGuest(ctx, c, &customer)
		})

	// signal handler for ensuring the customer is at least the given status. Used for invites and promoting an existing account.
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalEnsureMinimumStatus),
		func(c workflow.ReceiveChannel, _ bool) {
			signalEnsureMinimumStatus(ctx, c, &customer)
		})

	// signal handler for canceling account
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalCancelAccount),
		func(c workflow.ReceiveChannel, _ bool) {
			signalCancelAccount(ctx, c, &customer)
		})

	// handle Temporal Server cancellation requests
	selector.AddReceive(ctx.Done(),
		func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			logger.Info("Cancellation requested.")
			workflowCanceled = true
		})

	// query handler for status level, etc
	err := workflow.SetQueryHandler(ctx, QueryGetStatus,
		func() (GetStatusResponse, error) {
			return queryGetStatus(ctx, customer)
		})
	if err != nil {
		return fmt.Errorf("unable to register '%v' query handler: %w", QueryGetStatus, err)
	}

	// query handler for guest list
	err = workflow.SetQueryHandler(ctx, QueryGetGuests,
		func() ([]string, error) {
			return queryGetGuests(ctx, customer)
		})
	if err != nil {
		return fmt.Errorf("unable to register '%v' query handler: %w", QueryGetGuests, err)
	}

	// Block on everything. Continue-As-New on history length; size of activities in this workflow are small enough
	// that we'll hit the length thresholds well before any size threshold.
	logger.Info("Waiting for new signals")
	for customer.AccountActive && info.GetCurrentHistoryLength() < EventsThreshold && !workflowCanceled {
		selector.Select(ctx)

		if errSignal != nil {
			logger.Error("Unrecoverable error in handling a signal.", "Error", errSignal)
			return errSignal
		}
	}

	// here because of events threshold, but account still active? Continue-As-New
	if customer.AccountActive && !workflowCanceled {
		logger.Info("Account still active, but hit continue-as-new threshold; Continuing-As-New.", "Customer", customer.CustomerID)
		return workflow.NewContinueAsNewError(ctx, CustomerLoyaltyWorkflow, customer, false)
	}

	logger.Info("Loyalty workflow completed.", "Customer", customer, "WorkflowCanceled", workflowCanceled)
	if workflowCanceled {
		return ctx.Err()
	}
	return nil
}

// CustomerWorkflowID generates a Workflow ID based on the given customer ID.
func CustomerWorkflowID(customerID string) string {
	return "customer-" + customerID
}

func signalAddPoints(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) {
	logger := workflow.GetLogger(ctx)
	var activities Activities

	var pointsToAdd int
	c.Receive(ctx, &pointsToAdd)

	logger.Info("Adding points to customer account.", "PointsAdded", pointsToAdd)
	customer.LoyaltyPoints += pointsToAdd

	currentStatusOrd := customer.StatusLevel.Ordinal
	customer.StatusLevel = StatusLevelForPoints(customer.LoyaltyPoints)
	newStatusOrd := customer.StatusLevel.Ordinal

	statusChange := newStatusOrd - currentStatusOrd

	if statusChange > 0 {
		err := workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(emailPromoted, customer.StatusLevel.Name)).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity for status promotion.", "Error", err)
		}
	} else if statusChange < 0 {
		err := workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(emailDemoted, customer.StatusLevel.Name)).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity for status demotion.", "Error", err)
		}
	}
}

func signalInviteGuest(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) error {
	logger := workflow.GetLogger(ctx)
	var activities Activities

	var emailToSend string
	var guestID string
	c.Receive(ctx, &guestID)

	logger.Info("Checking to see if customer has enough status to allow for a guest invite.", "Customer", customer)
	if len(customer.Guests) < customer.StatusLevel.GuestsAllowed {
		logger.Info("Customer is allowed to invite guests. Attempting to invite.",
			"GuestID", guestID)

		guest := CustomerInfo{
			CustomerID:    guestID,
			StatusLevel:   customer.StatusLevel.Previous(),
			AccountActive: true,
		}

		customer.addGuest(guestID)
		err := workflow.ExecuteLocalActivity(ctx, activities.StartGuestWorkflow, guest).Get(ctx, nil)

		var actErr *temporal.ApplicationError
		errors.As(err, &actErr)
		if actErr != nil && actErr.Type() == "GuestAlreadyCanceledError" {
			emailToSend = emailGuestCanceled
		} else if err != nil {
			return fmt.Errorf("could not signal-with-start guest/child workflow for guest ID '%v': %w", guestID, err)
		} else {
			emailToSend = emailGuestInvited
		}
	} else {
		logger.Info("Customer does not have sufficient status to invite more guests.")
		emailToSend = emailInsufficientPoints
	}

	err := workflow.ExecuteActivity(ctx, activities.SendEmail, emailToSend).Get(ctx, nil)
	if err != nil {
		logger.Error("Error running SendEmail activity for guest invite.", "Error", err)
	}

	return nil
}

func signalEnsureMinimumStatus(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) {
	var activities Activities
	logger := workflow.GetLogger(ctx)

	var minStatusOrdinal int
	c.Receive(ctx, &minStatusOrdinal)

	if customer.StatusLevel.Ordinal < minStatusOrdinal {
		newStatus := StatusLevels[minStatusOrdinal]
		customer.StatusLevel = newStatus
		customer.LoyaltyPoints = newStatus.MinimumPoints

		emailBody := fmt.Sprintf(emailPromoted, newStatus.Name)
		err := workflow.ExecuteActivity(ctx, activities.SendEmail, emailBody).Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity for promotion.", "Error", err)
		}
	}
}

func signalCancelAccount(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) {
	logger := workflow.GetLogger(ctx)
	var activities Activities

	// nothing to receive, but need this to "handle" signal
	c.Receive(ctx, nil)

	customer.AccountActive = false
	err := workflow.ExecuteActivity(ctx, activities.SendEmail, emailCancelAccount).Get(ctx, nil)
	if err != nil {
		logger.Error("Error running SendEmail activity for account cancellation.", "Error", err)
	}

	logger.Info("Canceled account.", "CustomerID", customer.CustomerID)
}

func queryGetStatus(ctx workflow.Context, customer CustomerInfo) (GetStatusResponse, error) {
	logger := workflow.GetLogger(ctx)

	response := GetStatusResponse{
		StatusLevel:   *customer.StatusLevel,
		Points:        customer.LoyaltyPoints,
		AccountActive: customer.AccountActive,
	}
	logger.Info("Got response query.", "Customer", customer, "Response", response)

	return response, nil
}

func queryGetGuests(ctx workflow.Context, customer CustomerInfo) ([]string, error) {
	logger := workflow.GetLogger(ctx)
	guestIDs := customer.Guests

	logger.Info("Got guest list query.", "Guests", guestIDs)
	return guestIDs, nil
}
