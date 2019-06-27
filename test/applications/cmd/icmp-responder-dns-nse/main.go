package main

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
	"strings"
)

var version string

func main() {
	logrus.Info("Starting icmp-responder-dns-nse...")
	logrus.Infof("Version: %v", version)

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	ipamEndpoint := endpoint.NewIpamEndpoint(nil)
	routeAddr := endpoint.CreateRouteMutator([]string{"8.8.8.8/30"})
	if common.IsIPv6(ipamEndpoint.PrefixPool.GetPrefixes()[0]) {
		routeAddr = endpoint.CreateRouteMutator([]string{"2001:4860:4860::8888/126"})
	}

	composite := endpoint.NewCompositeEndpointBuilder().
		Append(
			endpoint.NewMonitorEndpoint(nil),
			endpoint.NewCustomFuncEndpoint("route", routeAddr),
			endpoint.NewCustomFuncEndpoint("dns", dnsConfigMutator),
			ipamEndpoint,
			endpoint.NewConnectionEndpoint(nil)).
		Build()

	nsmEndpoint, err := endpoint.NewNSMEndpoint(context.Background(), nil, composite)
	if err != nil {
		logrus.Fatalf("%v", err)
	}
	err = nsmEndpoint.Start()
	if err != nil {
		logrus.Fatalf("%v", err)
	}
	<-c
}

func dnsConfigMutator(c *connection.Connection) error {
	dnsIP := strings.Split(c.Context.IpContext.DstIpAddr, "/")[0]
	c.Context.DnsConfig = &connectioncontext.DNSConfig{
		DnsServerIps:    []string{dnsIP},
		ResolvesDomains: []string{"my.domain1", "my.domain2"},
		Prioritize:      true,
	}
	return nil
}