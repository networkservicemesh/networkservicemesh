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
	"encoding/binary"
	"net"

	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/pkg/tools"

	"github.com/ligato/networkservicemesh/controlplane/pkg/model/networkservice"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
)

func (ns *networkService) CompleteConnection(request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	err := ValidateNetworkServiceRequest(request)
	if err != nil {
		return nil, err
	}
	netns, _ := tools.GetCurrentNS()
	localMechanism := &common.LocalMechanism{
		Type: common.LocalMechanismType_KERNEL_INTERFACE,
		Parameters: map[string]string{
			nsmd.LocalMechanismParameterNetNsInodeKey: netns,
			// TODO: Fix this terrible hack using xid for getting a unique interface name
			nsmd.LocalMechanismParameterInterfaceNameKey: request.GetConnection().GetNetworkService() + request.GetConnection().GetConnectionId(),
		},
	}

	srcIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(srcIP, ns.nextIP)
	ns.nextIP = ns.nextIP + 1

	dstIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(dstIP, ns.nextIP)
	ns.nextIP = ns.nextIP + 3

	connectionContext := &networkservice.ConnectionContext{
		ConnectionContext: make(map[string]string),
	}

	connectionContext.ConnectionContext["src_ip"] = srcIP.String() + "/30"
	connectionContext.ConnectionContext["dst_ip"] = dstIP.String() + "/30"

	// TODO take into consideration LocalMechnism preferences sent in request

	connection := &networkservice.Connection{
		ConnectionId:      request.Connection.ConnectionId,
		NetworkService:    request.Connection.NetworkService,
		LocalMechanism:    localMechanism,
		ConnectionContext: connectionContext,
	}
	return connection, nil
}
