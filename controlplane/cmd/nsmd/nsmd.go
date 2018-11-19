package main

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"os"
	"os/signal"
	"strings"
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

	if err := nsmd.StartDataplaneRegistrarServer(model); err != nil {
		logrus.Fatalf("Error starting dataplane service: %+v", err)
		os.Exit(1)
	}

	crossConnectAddress := os.Getenv("NSMD_CROSS_CONNECT_ADDRESS")
	if strings.TrimSpace(crossConnectAddress) == "" {
		crossConnectAddress = "0.0.0.0:5007"
	}

	if err, monitorServer, _ := monitor_crossconnect_server.StartNSMCrossConnectServer(model, crossConnectAddress); err != nil {
		logrus.Fatalf("Error starting nsmd CrossConnect Server %+v", err)
		os.Exit(1)
	} else {
		defer monitorServer.Stop()
	}

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
