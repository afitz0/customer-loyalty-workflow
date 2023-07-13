package main

import (
	"context"
	"log"

	"go.temporal.io/sdk/client"

	"go.uber.org/zap/zapcore"

	wf "github.com/afitz0/customer-loyalty-workflow/go"
)

func main() {
	logger := wf.NewZapAdapter(wf.NewZapLogger(zapcore.DebugLevel))
	c, err := client.Dial(client.Options{
		Logger: logger,
	})
	if err != nil {
		log.Fatalln("Unable to create client.", err)
	}
	defer c.Close()

	customer := wf.CustomerInfo{
		CustomerID:    "123",
		Name:          "Customer",
		AccountActive: true,
	}
	workflowOptions := client.StartWorkflowOptions{
		ID:        wf.CustomerWorkflowID(customer.CustomerID),
		TaskQueue: wf.TaskQueue,
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, wf.CustomerLoyaltyWorkflow, customer)
	if err != nil {
		log.Fatalln("Unable to execute workflow.", err)
	}

	log.Println("Started workflow.", "WorkflowID", we.GetID(), "RunID", we.GetRunID())
}
