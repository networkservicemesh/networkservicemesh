package main

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
)

var version string

func main() {
	logrus.Info("Starting icmp-responder-dns-nse...")
	logrus.Infof("Version: %v", version)

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	ipamEndpoint := endpoint.NewIpamEndpoint(nil)
	routeAddr := makeRouteMutator([]string{"8.8.8.8/30"})
	if common.IsIPv6(ipamEndpoint.PrefixPool.GetPrefixes()[0]) {
		routeAddr = makeRouteMutator([]string{"2001:4860:4860::8888/126"})
	}
	dnsConfig := connectioncontext.DNSConfig{
		DnsServerIps:    []string{"172.16.1.2:53"},
		ResolvesDomains: []string{"my.own.google.com"},
	}
	err := dnsConfig.Validate()
	if err != nil {
		logrus.Fatalf("%v", err)
	}
	composite := endpoint.NewCompositeEndpointBuilder().
		Append(
			endpoint.NewMonitorEndpoint(nil),
			endpoint.NewCustomFuncEndpoint("route", routeAddr),
			endpoint.NewCustomFuncEndpoint("dns", func(c *connection.Connection) error {
				logrus.Infof("Injecting dns config: %v into: %v", dnsConfig, c)
				c.Context.DnsConfig = &dnsConfig
				return nil
			}),
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

//TODO: remove code duplication
func makeRouteMutator(routes []string) endpoint.ConnectionMutator {
	return func(c *connection.Connection) error {
		for _, r := range routes {
			c.Context.Routes = append(c.Context.Routes, &connectioncontext.Route{
				Prefix: r,
			})
		}
		return nil
	}
}
