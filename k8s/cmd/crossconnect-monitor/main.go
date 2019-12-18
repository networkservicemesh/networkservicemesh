package main

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

func main() {
	logrus.Info("Starting crossconnect-monitor...")
	utils.PrintAllEnv(logrus.StandardLogger())
	var wg sync.WaitGroup
	wg.Add(1)
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	go func() {
		<-c
		closing = true
		wg.Done()
	}()

	lookForNSMServers()

	wg.Wait()
}
