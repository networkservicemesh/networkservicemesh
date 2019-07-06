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
	"net"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

var version string

func parseFlags() (bool, bool, bool, bool) {
	dirty := flag.Bool("dirty", false,
		"will not delete itself from registry at the end")
	neighbors := flag.Bool("neighbors", false,
		"will set all available IpNeighbors to connection.Context")
	routes := flag.Bool("routes", false,
		"will set route 8.8.8.8/30 to connection.Context")
	update := flag.Bool("update", false,
		"will send update to local.Connection after some time")

	flag.Parse()

	return *dirty, *neighbors, *routes, *update
}

func main() {
	logrus.Info("Starting icmp-responder-nse...")
	logrus.Infof("Version: %v", version)
	dirty, neighbors, routes, update := parseFlags()

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	endpoints := []networkservice.NetworkServiceServer{
		endpoint.NewMonitorEndpoint(nil),
		endpoint.NewConnectionEndpoint(nil),
	}

	if neighbors {
		logrus.Infof("Adding neighbors endpoint to chain")
		endpoints = append(endpoints,
			endpoint.NewCustomFuncEndpoint("neighbor", ipNeighborMutator))
	}

	ipamEndpoint := endpoint.NewIpamEndpoint(nil)
	endpoints = append(endpoints, ipamEndpoint)
	if routes {
		prefixes := []string{"8.8.8.8/30"}
		if common.IsIPv6(ipamEndpoint.PrefixPool.GetPrefixes()[0]) {
			prefixes = []string{"2001:4860:4860::8888/126"}
		}
		endpoints = append(endpoints, endpoint.NewRoutesEndpoint(prefixes))
	}

	var monitorServer monitor.Server
	if update {
		logrus.Infof("Adding updating endpoint to chain")
		endpoints = append(endpoints,
			endpoint.NewCustomFuncEndpoint("update", func(*connection.Connection) error {
				go func() {
					<-time.After(10 * time.Second)
					updateConnections(monitorServer)
				}()
				return nil
			}))
	}

	composite := endpoint.NewCompositeEndpoint(endpoints...)

	nsmEndpoint, err := endpoint.NewNSMEndpoint(context.Background(), nil, composite)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	_ = nsmEndpoint.Start()
	if !dirty {
		defer func() { _ = nsmEndpoint.Delete() }()
	}

	// Capture signals to cleanup before exiting
	<-c
}

func makeRouteMutator(routes []string) endpoint.ConnectionMutator {
	return func(c *connection.Connection) error {
		for _, r := range routes {
			c.GetContext().GetIpContext().DstRoutes = append(c.GetContext().GetIpContext().GetDstRoutes(), &connectioncontext.Route{
				Prefix: r,
			})
		}
		return nil
	}
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
