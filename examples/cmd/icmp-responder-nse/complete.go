// Copyright (c) 2018 Cisco and/or its affiliates.
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
	"github.com/gogo/protobuf/proto"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/sirupsen/logrus"
	"net"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

func (ns *networkService) CompleteConnection(request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	err := request.IsValid()
	if err != nil {
		return nil, err
	}
	mechanism := &connection.Mechanism{
		Type: connection.MechanismType_KERNEL_INTERFACE,
		Parameters: map[string]string{
			connection.NetNsInodeKey: ns.netNS,
			// TODO: Fix this terrible hack using xid for getting a unique interface name
			connection.InterfaceNameKey: "nsm" + request.GetConnection().GetId(),
		},
	}

	//TODO: We need to somehow support IPv6.
	srcIP, dstIP, prefixes, err := ns.prefixPool.Extract(request.Connection.Id, connectioncontext.IpFamily_IPV4, request.Connection.Context.ExtraPrefixRequest...)
	if err != nil {
		return nil, err
	}

	// TODO take into consideration LocalMechnism preferences sent in request

	// Copy context to not miss any valuable parameters.
	context := proto.Clone(request.Connection.Context).(*connectioncontext.ConnectionContext)

	// Update source/dst IP's
	context.SrcIpAddr = srcIP.String()
	context.DstIpAddr = dstIP.String()

	//Add extra routes.
	context.Routes= []*connectioncontext.Route{
			&connectioncontext.Route{
				Prefix: "8.8.8.8/30",
			},
		}
	context.ExtraPrefixes= prefixes

	connection := &connection.Connection{
		Id:             request.GetConnection().GetId(),
		NetworkService: request.GetConnection().GetNetworkService(),
		Mechanism:      mechanism,
		Context: context,
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
					connection.Context.IpNeighbors = append(connection.Context.IpNeighbors,
						&connectioncontext.IpNeighbor{
							Ip:              addr.String(),
							HardwareAddress: iface.HardwareAddr.String(),
						},
					)
				}
			}
		}
	}

	err = connection.IsComplete()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logrus.Infof("NSE is complete response: %v", connection)
	return connection, nil
}
