package main

import (
	"sync"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

func main() {
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
