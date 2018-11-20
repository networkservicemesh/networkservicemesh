package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"github.com/sirupsen/logrus"
)

func main() {
	// Capture signals to cleanup before exiting
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	myModel := model.NewModel("127.0.0.1:5000")
	crossConnectAddress := "127.0.0.1:5007"

	err, grpcServer, _ := monitor_crossconnect_server.StartNSMCrossConnectServer(myModel, crossConnectAddress)

	if err != nil {
		logrus.Fatalf("Error starting crossconnect server: %s", err)
	}

	select {
	case <-c:
		logrus.Info("Closing Dataplane Registration")
		grpcServer.Stop()
	}
}
