// Copyright 2018 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/local"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

var version string

type icmpResponderFlags struct {
	isDirty      bool
	useNeighbors bool
	useRoutes    bool
	doUpdate     bool
	useDNS       bool
}

func parseFlags() icmpResponderFlags {
	dirty := flag.Bool("dirty", false,
		"will not delete itself from registry at the end")
	neighbors := flag.Bool("neighbors", false,
		"will set all available IpNeighbors to connection.Context")
	routes := flag.Bool("routes", false,
		"will set route 8.8.8.8/30 to connection.Context")
	update := flag.Bool("update", false,
		"will send update to local.Connection after some time")
	dns := flag.Bool("dns", false,
		fmt.Sprintf("means use dns mutator. DNS mutator using next env variables: %v, %v", SearchDomainsEnv.Name(), ServerIPsEnv.Name()))
	flag.Parse()
	return icmpResponderFlags{
		isDirty:      *dirty,
		useRoutes:    *routes,
		useNeighbors: *neighbors,
		doUpdate:     *update,
		useDNS:       *dns,
	}
}

func main() {
	logrus.Info("Starting icmp-responder-nse...")
	logrus.Infof("Version: %v", version)
	icmpFlags := parseFlags()

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	cb := endpoint.NewCompositeEndpointBuilder()
	monitorEndpoint := endpoint.NewMonitorEndpoint(nil)
	cb.Append(monitorEndpoint)
	if icmpFlags.useNeighbors {
		logrus.Infof("Adding neighbors endpoint to chain")
		cb.Append(
			endpoint.NewCustomFuncEndpoint("neighbor", ipNeighborMutator))
	}
	ipamEndpoint := endpoint.NewIpamEndpoint(nil)
	if icmpFlags.useRoutes {
		logrus.Infof("Adding routes endpoint to chain")
		routeAddr := endpoint.CreateRouteMutator([]string{"8.8.8.8/30"})
		if common.IsIPv6(ipamEndpoint.PrefixPool.GetPrefixes()[0]) {
			routeAddr = endpoint.CreateRouteMutator([]string{"2001:4860:4860::8888/126"})
		}
		cb.Append(endpoint.NewCustomFuncEndpoint("route", routeAddr))
	}
	if icmpFlags.useDNS {
		logrus.Info("Adding dns endpoint to chain")
		cb.Append(endpoint.NewCustomFuncEndpoint("dns", dnsMutator))
	}

	var monitorServer monitor.Server
	if icmpFlags.doUpdate {
		logrus.Infof("Adding updating endpoint to chain")
		cb.Append(
			endpoint.NewCustomFuncEndpoint("update", func(*connection.Connection) error {
				go func() {
					<-time.After(10 * time.Second)
					updateConnections(monitorServer)
				}()
				return nil
			}))
	}

	cb.Append(
		ipamEndpoint,
		endpoint.NewConnectionEndpoint(nil))

	nsmEndpoint, err := endpoint.NewNSMEndpoint(context.Background(), nil, cb.Build())
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	monitorServer = monitorEndpoint.GetOpaque(nil).(local.MonitorServer)

	_ = nsmEndpoint.Start()
	if !icmpFlags.isDirty {
		defer func() { _ = nsmEndpoint.Delete() }()
	}

	// Capture signals to cleanup before exiting
	<-c
}
func dnsMutator(c *connection.Connection) error {
	defaultIP := strings.Split(c.Context.IpContext.DstIpAddr, "/")[0]
	c.Context.DnsContext = &connectioncontext.DNSContext{
		Configs: []*connectioncontext.DNSConfig{
			{
				DnsServerIps:  ServerIPsEnv.GetOrDefaultStringListValue([]string{defaultIP}),
				SearchDomains: SearchDomainsEnv.GetOrDefaultStringListValue([]string{"icmp.app"}),
			},
		},
	}
	return nil
}

func ipNeighborMutator(c *connection.Connection) error {
	addrs, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, iface := range addrs {
		adrs, err := iface.Addrs()
		if err != nil {
			logrus.Error(err)
			continue
		}

		for _, a := range adrs {
			addr, _, _ := net.ParseCIDR(a.String())
			if !addr.IsLoopback() {
				c.GetContext().GetIpContext().IpNeighbors = append(c.GetContext().GetIpContext().GetIpNeighbors(),
					&connectioncontext.IpNeighbor{
						Ip:              addr.String(),
						HardwareAddress: iface.HardwareAddr.String(),
					},
				)
			}
		}
	}
	return nil
}

func updateConnections(monitorServer monitor.Server) {
	for _, entity := range monitorServer.Entities() {
		localConnection := proto.Clone(entity.(*connection.Connection)).(*connection.Connection)
		localConnection.GetContext().GetIpContext().ExcludedPrefixes =
			append(localConnection.GetContext().GetIpContext().GetExcludedPrefixes(), "255.255.255.255/32")

		monitorServer.Update(localConnection)
	}
}
