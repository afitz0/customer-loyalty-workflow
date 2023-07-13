package loyalty

import (
	"context"
	"errors"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
)

type Activities struct {
	Client client.Client
}

func (*Activities) SendEmail(ctx context.Context, body string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending email.", "Contents", body)
	return nil
}

func (a *Activities) StartGuestWorkflow(ctx context.Context, guest CustomerInfo) error {
	logger := activity.GetLogger(ctx)

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue:             TaskQueue,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}

	logger.Info("Starting and signaling guest workflow.", "GuestID", guest.CustomerID)
	_, err := a.Client.SignalWithStartWorkflow(ctx, CustomerWorkflowID(guest.CustomerID),
		SignalEnsureMinimumStatus, guest.StatusLevel,
		workflowOptions, CustomerLoyaltyWorkflow, guest)
	target := &serviceerror.WorkflowExecutionAlreadyStarted{}
	if errors.As(err, &target) {
		return &GuestAlreadyCanceledError{msg: "Guest account cannot be recreated from a closed status."}
	} else if err != nil {
		return err
	}

	return nil
}
