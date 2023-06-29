package starter

import (
	"fmt"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/temporal"
	"reflect"
	"time"

	"go.temporal.io/sdk/workflow"
)

func CustomerLoyaltyWorkflow(ctx workflow.Context, customer CustomerInfo) (err error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Loyalty workflow started", "CustomerInfo", customer)

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

	validateCustomerInfo(customer)

	if info.ContinuedExecutionRunID == "" {
		err = workflow.ExecuteActivity(ctx, activities.SendEmail,
			fmt.Sprintf(EmailWelcome, StatusTiers[customer.StatusLevel].Name)).
			Get(ctx, nil)
		if err != nil {
			logger.Error("Error running SendEmail activity", "Error", err)
			return err
		}
	}

	// signal handler for adding points
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalAddPoints),
		func(c workflow.ReceiveChannel, _ bool) {
			var pointsToAdd int
			c.Receive(ctx, &pointsToAdd)

			logger.Info("Adding points to customer account", "PointsAdded", pointsToAdd)
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
					logger.Error("Error running SendEmail activity", "Error", err)
					errSignal = err
				}
			}
		})

	// signal handler for adding guest
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalInviteGuest),
		func(c workflow.ReceiveChannel, _ bool) {
			var emailToSend string

			if len(customer.Guests) < StatusTiers[customer.StatusLevel].GuestsAllowed {
				var guestId string
				c.Receive(ctx, &guestId)

				var status GetStatusResponse
				err := workflow.ExecuteActivity(ctx, activities.QueryCustomerStatus, guestId).
					Get(ctx, &status)
				if err != nil {
					logger.Error("Error getting potential guest customer status", "Error", err)
					errSignal = err
					return
				}

				logger.Info("Got status response", "Response", status)
				guest := CustomerInfo{
					CustomerId:    guestId,
					LoyaltyPoints: status.Points,
					StatusLevel:   status.StatusLevel,
					AccountActive: status.AccountActive,
				}

				if guest.AccountActive {
					previousTier := StatusTiers[min(customer.StatusLevel-1, 0)]
					err = startGuestWorkflow(ctx, guest, previousTier)
					if err != nil {
						logger.Error("Could not start guest/child workflow.",
							"Error", err,
							"Error Type", reflect.ValueOf(err).Type())
						errSignal = err
						return
					}

					if _, ok := err.(*serviceerror.WorkflowExecutionAlreadyStarted); ok {
						err = workflow.SignalExternalWorkflow(ctx,
							fmt.Sprintf(CustomerWorkflowIdFormat, guestId),
							"",
							SignalEnsureMinimumStatus,
							previousTier).Get(ctx, nil)
						if err != nil {
							errSignal = err
							logger.Error("Could not signal child to ensure min status", err)
							return
						}
					} else {
						customer.Guests = append(customer.Guests, guest.CustomerId)
						emailToSend = EmailGuestInvited
					}
				} else {
					emailToSend = EmailGuestCanceled
				}
			} else {
				emailToSend = EmailInsufficientPoints
			}

			err = workflow.ExecuteActivity(ctx, activities.SendEmail, emailToSend).Get(ctx, nil)
			if err != nil {
				logger.Error("Error running SendEmail activity", "Error", err)
				errSignal = err
			}
		})

	// signal handler for ensuring the customer is at least the given status. Used for invites and promoting an existing account.
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalEnsureMinimumStatus),
		func(c workflow.ReceiveChannel, _ bool) {
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
					logger.Error("Error running SendEmail activity", "Error", err)
					errSignal = err
				}
			}
		})

	// signal handler for canceling account
	selector.AddReceive(workflow.GetSignalChannel(ctx, SignalCancelAccount),
		func(c workflow.ReceiveChannel, _ bool) {
			// nothing to receive, but need this to "handle" signal
			c.Receive(ctx, nil)

			customer.AccountActive = false
			err = workflow.ExecuteActivity(ctx, activities.SendEmail, EmailCancelAccount).Get(ctx, nil)
			if err != nil {
				logger.Error("Error running SendEmail activity", "Error", err)
				errSignal = err
				return
			}

			logger.Info("Canceled account", "CustomerID", customer.CustomerId)
		})

	// query handler for status level
	// Set up the Query handler for the response
	err = workflow.SetQueryHandler(ctx, QueryGetStatus,
		func() (GetStatusResponse, error) {
			logger.Info("Got status query", "Customer", customer)
			status := GetStatusResponse{
				StatusLevel:   customer.StatusLevel,
				Tier:          StatusTiers[customer.StatusLevel],
				Points:        customer.LoyaltyPoints,
				AccountActive: customer.AccountActive,
			}

			return status, nil
		})

	err = workflow.SetQueryHandler(ctx, QueryGetGuests,
		func() ([]string, error) {
			logger.Info("Sending back guest list", "Guests", customer.Guests)
			return customer.Guests, nil
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

func startGuestWorkflow(ctx workflow.Context, guest CustomerInfo, minStatus StatusTier) error {
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		WorkflowID:            fmt.Sprintf(CustomerWorkflowIdFormat, guest.CustomerId),
		ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}
	ctx = workflow.WithChildOptions(ctx, childWorkflowOptions)

	childWorkflowFuture := workflow.ExecuteChildWorkflow(ctx, CustomerLoyaltyWorkflow, guest)
	return childWorkflowFuture.
		SignalChildWorkflow(ctx, SignalEnsureMinimumStatus, minStatus).
		Get(ctx, nil)
}

func validateCustomerInfo(customer CustomerInfo) {
	if customer.StatusLevel >= len(StatusTiers) {
		customer.StatusLevel = len(StatusTiers) - 1
	}
	if customer.StatusLevel < 0 {
		customer.StatusLevel = 0
	}

	if customer.Guests == nil {
		customer.Guests = make([]string, StatusTiers[customer.StatusLevel].GuestsAllowed)
	}
}
