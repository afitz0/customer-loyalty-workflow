package starter

import (
	"context"
	"go.temporal.io/sdk/activity"
)

type Activities struct{}

func (a *Activities) SendEmail(ctx context.Context, body string) (err error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending email", "Contents", body)
	logger.Info("Activity name", "name", activity.GetInfo(ctx).ActivityType.Name)
	return nil
}
