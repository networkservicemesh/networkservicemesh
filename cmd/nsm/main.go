package main

import (
	"github.com/ligato/networkservicemesh/nsmdp"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	var wg sync.WaitGroup

	log.SetFlags(log.Flags() | log.Lshortfile)
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
		log.Printf("Received signal \"%v\", shutting down.", s)
		dp.Stop()
	}()

	wg.Wait()
	log.Println("Stopping NSM")
}
