package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
)

func main() {

	composite := endpoint.NewCompositeEndpoint(
		endpoint.NewMonitorEndpoint(nil),
		endpoint.NewIpamEndpoint(nil),
		endpoint.NewConnectionEndpoint(nil))

	nsmEndpoint, err := endpoint.NewNSMEndpoint(nil, nil, composite)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	nsmEndpoint.Start()

	c := make(chan os.Signal)
	signal.Notify(c,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	<-c
}