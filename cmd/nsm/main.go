package main

import (
	"github.com/ligato/networkservicemesh/nsmdp"
	"log"
	"os"
	"os/signal"
	"syscall"
	"sync"
)

var wg sync.WaitGroup

func main() {
	log.Println("Starting NSM")

	dp := nsmdp.NewNSMDevicePlugin()
	dp.Serve()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	wg.Add(1)

	go func() {
		defer wg.Done()
		s := <-sigChan
		log.Println("Received signal \"%v\", shutting down.", s)
		dp.Stop()
	}()

	log.Println("Stopping NSM")
	wg.Wait()
}
