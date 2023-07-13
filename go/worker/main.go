package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"go.uber.org/zap/zapcore"

	wf "github.com/afitz0/customer-loyalty-workflow/go"
	"github.com/afitz0/customer-loyalty-workflow/go/zapadapter"
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

	w := worker.New(c, wf.TaskQueue, worker.Options{})

	a := &wf.Activities{
		Client: c,
	}
	w.RegisterWorkflow(wf.CustomerLoyaltyWorkflow)
	w.RegisterActivity(a)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker.", err)
	}
}
