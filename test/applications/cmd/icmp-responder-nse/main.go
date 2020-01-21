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
	"net"
	"time"

	connectionMonitor "github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"
	"github.com/networkservicemesh/networkservicemesh/utils"

	"github.com/networkservicemesh/networkservicemesh/test/applications/cmd/icmp-responder-nse/flags"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

var version string

func main() {
	logrus.Info("Starting icmp-responder-nse...")
	logrus.Infof("Version: %v", version)
	utils.PrintAllEnv(logrus.StandardLogger())
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	flags := flags.ParseFlags()

	configuration := common.FromEnv()
	configuration.MechanismType = "SRIOV_KERNEL_INTERFACE"

	endpoints := []networkservice.NetworkServiceServer{
		endpoint.NewMonitorEndpoint(configuration),
		endpoint.NewConnectionEndpoint(configuration),
	}

	if flags.Neighbors {
		logrus.Infof("Adding neighbors endpoint to chain")
		endpoints = append(endpoints,
			endpoint.NewCustomFuncEndpoint("neighbor", ipNeighborMutator))
	}

	ipamEndpoint := endpoint.NewIpamEndpoint(configuration)
	endpoints = append(endpoints, ipamEndpoint)

	routeAddr := endpoint.CreateRouteMutator([]string{"8.8.8.8/30"})
	prefixes := ipamEndpoint.PrefixPool.GetPrefixes()
	if len(prefixes) > 0 && common.IsIPv6(prefixes[0]) {
		routeAddr = endpoint.CreateRouteMutator([]string{"2001:4860:4860::8888/126"})
	}
	if flags.Routes {
		logrus.Infof("Adding routes endpoint to chain")
		endpoints = append(endpoints, endpoint.NewCustomFuncEndpoint("route", routeAddr))
	}

	if flags.DNS {
		logrus.Info("Adding dns endpoint to chain")
		if len(ServerIPsEnv.GetStringListValueOrDefault("")) == 1 && ServerIPsEnv.GetStringListValueOrDefault("")[0] == "" {
			endpoints = append(endpoints, endpoint.NewAddDnsConfigDstIp(SearchDomainsEnv.GetStringListValueOrDefault("icmp.app")...))
		} else {
			endpoint.NewAddDNSConfigs(&connectioncontext.DNSConfig{
				DnsServerIps:  ServerIPsEnv.GetStringListValueOrDefault(""),
				SearchDomains: SearchDomainsEnv.GetStringListValueOrDefault("icmp.app"),
			})
		}
	}

	podName := endpoint.CreatePodNameMutator()
	endpoints = append(endpoints, endpoint.NewCustomFuncEndpoint("podName", podName))

	if flags.Update {
		logrus.Infof("Adding updating endpoint to chain")
		endpoints = append(endpoints,
			endpoint.NewCustomFuncEndpoint("update", func(ctx context.Context, conn *connection.Connection) error {
				monitorServer := endpoint.MonitorServer(ctx)
				logrus.Infof("Delaying 5 seconds before send update event.")
				go func() {
					for i := 0; i < 10; i++ {
						updateConnections(ctx, monitorServer)
						logrus.Infof("Update event %v sended.", i)
						<-time.After(5 * time.Second)
					}
				}()
				return nil
			}))
	}

	composite := endpoint.NewCompositeEndpoint(endpoints...)

	nsmEndpoint, err := endpoint.NewNSMEndpoint(context.Background(), configuration, composite)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	_ = nsmEndpoint.Start()
	if !flags.Dirty {
		defer func() { _ = nsmEndpoint.Delete() }()
	}

	// Capture signals to cleanup before exiting
	<-c
}

func ipNeighborMutator(ctc context.Context, c *connection.Connection) error {
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

func updateConnections(ctx context.Context, monitorServer connectionMonitor.MonitorServer) {
	for _, entity := range monitorServer.Entities() {
		localConnection := proto.Clone(entity.(*connection.Connection)).(*connection.Connection)
		localConnection.GetContext().GetIpContext().ExcludedPrefixes =
			append(localConnection.GetContext().GetIpContext().GetExcludedPrefixes(), "255.255.255.255/32")

		monitorServer.Update(ctx, localConnection)
	}
}
