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
	model := model.NewModel()
	defer nsmd.StopRegistryClient()

	// registryClient, err := nsmd.RegistryClient()
	// if err != nil {
	// 	logrus.Fatalf("Error Connecting to Registry Service: %+v", err)
	// 	os.Exit(1)
	// }
	if err := nsmd.StartDataplaneRegistrarServer(model); err != nil {
		logrus.Fatalf("Error starting dataplane service: %+v", err)
		os.Exit(1)
	}
	// if err := nsmd.StartEndpointServer(model, registryClient); err != nil {
	// 	logrus.Fatalf("Error starting endpoint service: %+v", err)
	// 	os.Exit(1)
	// }
	if err := nsmd.StartNSMServer(model); err != nil {
		logrus.Fatalf("Error starting nsmd service: %+v", err)
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
