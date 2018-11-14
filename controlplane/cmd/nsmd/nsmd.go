package main

import (
	"io/ioutil"
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
	model := model.NewModel(nsmUrl)
	defer nsmd.StopRegistryClient()

	// spin up registry client
	_, err := nsmd.RegistryClient()
	if err != nil {
		logrus.Fatalf("Error starting registry client")
	}

	if err := nsmd.StartDataplaneRegistrarServer(model); err != nil {
		logrus.Fatalf("Error starting dataplane service: %+v", err)
	}

	if err := nsmd.StartNSMServer(model); err != nil {
		logrus.Fatalf("Error starting nsmd service: %+v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		wg.Done()
	}()
	err = ioutil.WriteFile("/tmp/online", []byte("online"), 555)
	if err != nil {
		logrus.Fatalf("Error writing /tmp/online readiness check", err)
	}
	wg.Wait()
}
