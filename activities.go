package starter

import (
	"context"
	"fmt"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"

	"go.uber.org/zap/zapcore"

	"starter/zapadapter"
)

type Activities struct{}

func (a *Activities) SendEmail(ctx context.Context, body string) (err error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending email", "Contents", body)
	return nil
}

func (a *Activities) QueryCustomerStatus(ctx context.Context, customerId string) (GetStatusResponse, error) {
	logger := zapadapter.NewZapAdapter(zapadapter.NewZapLogger(zapcore.DebugLevel))
	c, err := client.Dial(client.Options{
		Logger: logger,
	})

	resp, err := c.QueryWorkflow(ctx,
		fmt.Sprintf(CustomerWorkflowIdFormat, customerId),
		"",
		"getStatus")
	if _, isNotFound := err.(*serviceerror.NotFound); isNotFound {
		// If this account doesn't exist yet, it needs to become active.
		return GetStatusResponse{
			AccountActive: true,
		}, nil
	}

	var status GetStatusResponse
	err = resp.Get(&status)
	return status, err
}
