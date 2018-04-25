package main

import (
	"github.com/ligato/networkservicemesh/nsmdp"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
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
	select {
	case s := <-sigChan:
		log.Printf("Received signal \"%v\", shutting down.", s)
		dp.Stop()
	}

}
