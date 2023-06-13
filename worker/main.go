package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

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
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	w := worker.New(c, "temporal-starter", worker.Options{})

	a := &starter.Activities{}
	w.RegisterWorkflow(starter.Workflow)
	w.RegisterActivity(a)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
