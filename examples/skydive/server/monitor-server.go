package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"

	"github.com/ligato/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	// Capture signals to cleanup before exiting
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	crossConnectAddress := "127.0.0.1:5007"

	crossConnectServer := monitor_crossconnect_server.NewMonitorCrossConnectServer()
	listener, err := net.Listen("tcp", crossConnectAddress)
	if err != nil {
		logrus.Fatalf("Error listening on %s: %s", crossConnectAddress, err)
	}
	grpcServer := grpc.NewServer()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, crossConnectServer)
	go func() {
		err := grpcServer.Serve(listener)
		if err != nil {
			logrus.Fatalf("Error serving MonitorCrossConnect: %s", err)
		}
	}()

	select {
	case <-c:
		logrus.Info("Closing Dataplane Registration")
		grpcServer.Stop()
	}
}
