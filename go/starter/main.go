package main

import (
	"context"
	"fmt"
	"log"

	"go.temporal.io/sdk/client"

	"go.uber.org/zap/zapcore"

	wf "github.com/afitz0/customer-loyalty-workflow"
	"github.com/afitz0/customer-loyalty-workflow/common"
	"github.com/afitz0/customer-loyalty-workflow/zapadapter"
)

func main() {
	logger := zapadapter.NewZapAdapter(zapadapter.NewZapLogger(zapcore.DebugLevel))
	c, err := client.Dial(client.Options{
		Logger: logger,
	})
	if err != nil {
		log.Fatalln("Unable to create client.", err)
	}
	defer c.Close()

	customer := wf.CustomerInfo{
		CustomerID:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		Guests:        map[string]struct{}{},
		AccountActive: true,
	}
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf(common.CustomerWorkflowIDFormat, customer.CustomerID),
		TaskQueue: common.TaskQueue,
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, wf.CustomerLoyaltyWorkflow, customer)
	if err != nil {
		log.Fatalln("Unable to execute workflow.", err)
	}

	log.Println("Started workflow.", "WorkflowID", we.GetID(), "RunID", we.GetRunID())
}
