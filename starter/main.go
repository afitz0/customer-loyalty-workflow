package main

import (
	"context"
	"fmt"
	"log"

	"go.temporal.io/sdk/client"

	"go.uber.org/zap/zapcore"

	"starter"
	"starter/zapadapter"
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

	customer := starter.CustomerInfo{
		CustomerId:    "123",
		LoyaltyPoints: 0,
		StatusLevel:   0,
		Name:          "Customer",
		Guests:        map[string]struct{}{},
		AccountActive: true,
	}
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf(starter.CustomerWorkflowIdFormat, customer.CustomerId),
		TaskQueue: starter.TaskQueue,
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, starter.CustomerLoyaltyWorkflow, customer)
	if err != nil {
		log.Fatalln("Unable to execute workflow.", err)
	}

	log.Println("Started workflow.", "WorkflowID", we.GetID(), "RunID", we.GetRunID())
}
