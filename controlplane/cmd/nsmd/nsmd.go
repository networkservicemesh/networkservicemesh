package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/sirupsen/logrus"
)

func main() {

	apiRegistry := nsmd.NewApiRegistry()
	serviceRegistry := nsmd.NewServiceRegistry()

	model := model.NewModel() // This is TCP gRPC server uri to access this NSMD via network.

	defer serviceRegistry.Stop()

	if err := nsmd.StartDataplaneRegistrarServer(model); err != nil {
		logrus.Fatalf("Error starting dataplane service: %+v", err)
		os.Exit(1)
	}

	if err := nsmd.StartNSMServer(model, serviceRegistry, apiRegistry); err != nil {
		logrus.Fatalf("Error starting nsmd service: %+v", err)
		os.Exit(1)
	}

	if err := nsmd.StartAPIServer(model, apiRegistry, serviceRegistry); err != nil {
		logrus.Fatalf("Error starting nsmd api service: %+v", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		wg.Done()
	}()
	wg.Wait()
}
