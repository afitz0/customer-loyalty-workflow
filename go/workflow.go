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
	EmailWelcome            = "Welcome to our loyalty program! You're starting out at '%v' status."
	EmailGuestCanceled      = "Sorry, your guest has already canceled their account."
	EmailGuestInvited       = "Congratulations! Your guest has been invited!"
	EmailInsufficientPoints = "Sorry, you need to earn more points to invite more guests!"
	EmailPromoted           = "Congratulations! You've been promoted to '%v' status!"
	EmailDemoted            = "Unfortunately, you've lost enough points to bump you down to '%v' status. 😞"
	EmailCancelAccount      = "Sorry to see you go!"
)

type GuestAlreadyCanceledError struct {
	msg string
}

func (e *GuestAlreadyCanceledError) Error() string { return e.msg }

func CustomerLoyaltyWorkflow(ctx workflow.Context, customer CustomerInfo) (err error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Loyalty workflow started.", "CustomerInfo", customer)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
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
	activities := Activities{}
	var errSignal error

	logger.Debug("Got workflow info.", "WorkflowID", info.WorkflowExecution.ID)

	logger.Info("Validating customer info.")
	validateCustomerInfo(&customer)

	if info.ContinuedExecutionRunID == "" {
		err = workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(EmailWelcome, customer.StatusLevel.Name)).
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
	for customer.AccountActive && info.GetCurrentHistoryLength() < EventsThreshold {
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

// CustomerWorkflowID generates a Workflow ID based on the given customer ID.
func CustomerWorkflowID(customerID string) string {
	return fmt.Sprintf("customer-%v", customerID)
}

func validateCustomerInfo(customer *CustomerInfo) {
	if customer.StatusLevel == nil {
		customer.StatusLevel = StatusLevelForPoints(customer.LoyaltyPoints)
	}
	if len(customer.Guests) == 0 {
		customer.Guests = make([]string, customer.StatusLevel.GuestsAllowed)
	}
}

func addGuestToCustomer(customer *CustomerInfo, guestID string) error {
	guestExists := false
	for _, g := range customer.Guests {
		guestExists = g == guestID
	}
	// replace first empty slot, or append if none
	if !guestExists {
		added := false
		for i, g := range customer.Guests {
			if len(g) == 0 {
				customer.Guests[i] = guestID
				added = true
				break
			}
		}
		if !added {
			customer.Guests = append(customer.Guests, guestID)
		}
	}

	return nil
}

func signalAddPoints(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) error {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

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
			fmt.Sprintf(EmailPromoted, customer.StatusLevel.Name)).
			Get(ctx, nil)
		if err != nil {
			return fmt.Errorf("error running SendEmail activity for status promotion: %w", err)
		}
	} else if statusChange < 0 {
		err := workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(EmailDemoted, customer.StatusLevel.Name)).
			Get(ctx, nil)
		if err != nil {
			return fmt.Errorf("error running SendEmail activity for status demotion: %w", err)
		}
	}

	return nil
}

func signalInviteGuest(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) error {
	logger := workflow.GetLogger(ctx)
	activities := Activities{}

	var emailToSend string
	var guestID string
	c.Receive(ctx, &guestID)

	logger.Info("Checking to see if customer has enough status to allow for a guest invite.", "Customer", customer)
	nonEmptyGuests := 0
	for _, v := range customer.Guests {
		if len(v) > 0 {
			nonEmptyGuests++
		}
	}
	if nonEmptyGuests < customer.StatusLevel.GuestsAllowed {
		logger.Info("Customer is allowed to invite guests. Attempting to invite.",
			"GuestID", guestID)

		guest := CustomerInfo{
			CustomerID:    guestID,
			StatusLevel:   customer.StatusLevel.Previous(),
			AccountActive: true,
		}

		err := addGuestToCustomer(customer, guestID)
		if err != nil {
			return fmt.Errorf("unable to add guest '%v' to customer's guest list: %w", guestID, err)
		}

		err = workflow.ExecuteLocalActivity(ctx, activities.StartGuestWorkflow, guest).Get(ctx, nil)

		t := &GuestAlreadyCanceledError{}
		if errors.As(err, &t) {
			emailToSend = EmailGuestCanceled
		} else if err != nil {
			return fmt.Errorf("could not signal-with-start guest/child workflow for guest ID '%v': %w", guestID, err)
		} else {
			emailToSend = EmailGuestInvited
		}
	} else {
		logger.Info("Customer does not have sufficient status to invite more guests.")
		emailToSend = EmailInsufficientPoints
	}

	err := workflow.ExecuteActivity(ctx, activities.SendEmail, emailToSend).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("error running SendEmail activity: %w", err)
	}

	return nil
}

func signalEnsureMinimumStatus(ctx workflow.Context, c workflow.ReceiveChannel, customer *CustomerInfo) error {
	activities := Activities{}

	var minStatus StatusLevel
	c.Receive(ctx, &minStatus)

	minOrd := minStatus.Ordinal
	currentOrd := customer.StatusLevel.Ordinal

	if currentOrd < minOrd {
		customer.StatusLevel = &minStatus
		customer.LoyaltyPoints = minStatus.MinimumPoints

		emailBody := fmt.Sprintf(EmailPromoted, minStatus.Name)
		err := workflow.ExecuteActivity(ctx, activities.SendEmail, emailBody).Get(ctx, nil)
		if err != nil {
			return fmt.Errorf("error running SendEmail activity: %w", err)
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
	err := workflow.ExecuteActivity(ctx, activities.SendEmail, EmailCancelAccount).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("error running SendEmail activity: %w", err)
	}

	logger.Info("Canceled account.", "CustomerID", customer.CustomerID)
	return nil
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
