package starter

import (
	"fmt"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
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

			// Check if current points warrants new status tier
			if customer.StatusLevel < len(StatusTiers) {
				if customer.LoyaltyPoints >= StatusTiers[min(customer.StatusLevel+1, len(StatusTiers))].MinimumPoints {
					// TODO promote to next level
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
				err := workflow.ExecuteActivity(ctx, activities.QueryCustomerStatus, guestId).Get(ctx, &status)
				if err != nil {
					logger.Error("Error getting potential guest customer status", "Error", err)
					errSignal = err
					return
				}

				guest := CustomerInfo{
					CustomerId:    guestId,
					LoyaltyPoints: status.Points,
					StatusLevel:   status.StatusLevel,
					AccountActive: status.AccountActive,
				}

				switch status.AccountActive {
				case false:
					emailToSend = EmailGuestCanceled
				case true:
					previousTier := StatusTiers[min(customer.StatusLevel-1, 0)]
					err = startGuestWorkflow(ctx, guest, previousTier)
					if err != nil {
						logger.Error("Could not start guest/child workflow.", "Error", err)
						errSignal = err
						return
					}

					emailToSend = EmailGuestInvited
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
			for StatusTiers[customer.StatusLevel].MinimumPoints < minStatus.MinimumPoints {
				customer.StatusLevel++
				promoted = true
			}

			if promoted {
				emailBody := fmt.Sprintf("Congratulations! You've been promoted to '%v' status!",
					minStatus.Name)
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
			err = workflow.ExecuteActivity(ctx, activities.SendEmail, "Sorry to see you go!").Get(ctx, nil)
			if err != nil {
				logger.Error("Error running SendEmail activity", "Error", err)
				errSignal = err
			}
		})

	// query handler for status level
	// Set up the Query handler for the response
	err = workflow.SetQueryHandler(ctx, QueryGetStatus,
		func() (GetStatusResponse, error) {
			status := GetStatusResponse{
				StatusLevel:   customer.StatusLevel,
				Tier:          StatusTiers[customer.StatusLevel],
				Points:        customer.LoyaltyPoints,
				AccountActive: customer.AccountActive,
			}

			return status, nil
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

	logger.Info("Loyalty workflow completed.")
	return nil
}

func startGuestWorkflow(ctx workflow.Context, guest CustomerInfo, minStatus StatusTier) error {
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		WorkflowID:        fmt.Sprintf(CustomerWorkflowIdFormat, guest.CustomerId),
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
	}
	ctx = workflow.WithChildOptions(ctx, childWorkflowOptions)

	// TODO add child to guest list

	childWorkflowFuture := workflow.ExecuteChildWorkflow(ctx, CustomerLoyaltyWorkflow, guest)

	// Wait for the Child Workflow Execution to spawn
	//var childWE workflow.Execution
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
