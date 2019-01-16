// Copyright 2018, 2019 VMware, Inc.
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

package composite

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/sdk/common"
	"github.com/ligato/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
)

type IpamCompositeEndpoint struct {
	endpoint.BaseCompositeEndpoint
	nextIP uint32
}

func (ice *IpamCompositeEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if ice.GetNext() == nil {
		err := fmt.Errorf("IPAM needs next")
		logrus.Errorf("%v", err)
		return nil, err
	}

	newConnection, err := ice.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	// Force the connection src/dst IPs if configured
	srcIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(srcIP, ice.nextIP)
	ice.nextIP = ice.nextIP + 1

	dstIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(dstIP, ice.nextIP)
	ice.nextIP = ice.nextIP + 3

	newConnection.Context = &connectioncontext.ConnectionContext{
		SrcIpAddr: srcIP.String() + "/30",
		DstIpAddr: dstIP.String() + "/30",
		Routes: []*connectioncontext.Route{
			&connectioncontext.Route{
				Prefix: "8.8.8.8/30",
			},
		},
	}

	addrs, err := net.Interfaces()
	if err == nil {
		for _, iface := range addrs {
			adrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, a := range adrs {
				addr, _, _ := net.ParseCIDR(a.String())
				if addr.String() != "127.0.0.1" {
					newConnection.Context.IpNeighbors = append(newConnection.Context.IpNeighbors,
						&connectioncontext.IpNeighbor{
							Ip:              addr.String(),
							HardwareAddress: iface.HardwareAddr.String(),
						},
					)
				}
			}
		}
	}

	err = newConnection.IsComplete()
	if err != nil {
		logrus.Errorf("New connection is not complete: %v", err)
		return nil, err
	}

	logrus.Infof("IPAM completed on connection: %v", newConnection)
	return newConnection, nil
}

func (ice *IpamCompositeEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if ice.GetNext() != nil {
		return ice.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

func NewIpamCompositeEndpoint(configuration *common.NSConfiguration) *IpamCompositeEndpoint {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	rand.Seed(time.Now().UTC().UnixNano())

	self := &IpamCompositeEndpoint{
		nextIP: common.Ip2int(net.ParseIP(configuration.IPAddress)),
	}
	self.SetSelf(self)

	return self
}
