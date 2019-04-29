package main

import (
	"time"

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

	_ = nsmEndpoint.Start()

	for {
		time.Sleep(1 * time.Second)
	}
}
