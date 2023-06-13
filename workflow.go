package starter

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func Workflow(ctx workflow.Context, greeting string, name string) (string, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("Starter workflow started", "greeting", greeting, "name", name)

	var a *Activities
	var result string
	err := workflow.ExecuteActivity(ctx, a.Activity, greeting, name).Get(ctx, &result)
	if err != nil {
		logger.Error("Activity failed.", "Error", err)
		return "", err
	}

	logger.Info("Starter workflow completed.", "result", result)
	return result, nil
}
