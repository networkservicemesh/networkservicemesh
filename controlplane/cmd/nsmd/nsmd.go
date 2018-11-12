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
	// TODO add a real url
	nsmUrl := "127.0.0.1:5000"
	nsmModel := model.NewModel(nsmUrl)
	defer nsmd.StopRegistryClient()

	if err := nsmd.StartDataplaneRegistrarServer(nsmModel); err != nil {
		logrus.Fatalf("Error starting dataplane service: %+v", err)
		os.Exit(1)
	}

	if err := nsmd.StartNSMServer(nsmModel); err != nil {
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
