package loyalty

import (
	"context"
	"go.temporal.io/sdk/activity"
)

type Activities struct{}

func (*Activities) SendEmail(ctx context.Context, body string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending email.", "Contents", body)
	return nil
}
