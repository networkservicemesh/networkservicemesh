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

	"github.com/sirupsen/logrus"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

func (ns *vppagentNetworkService) CompleteConnection(request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	err := request.IsValid()
	if err != nil {
		return nil, err
	}
	mechanism, err := connection.NewMechanism(connection.MechanismType_MEM_INTERFACE, "nsm"+request.GetConnection().GetId(), "")
	if err != nil {
		return nil, err
	}

	srcIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(srcIP, ns.nextIP)
	ns.nextIP = ns.nextIP + 1

	dstIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(dstIP, ns.nextIP)
	ns.nextIP = ns.nextIP + 3

	connectionContext := make(map[string]string)

	connectionContext["src_ip"] = srcIP.String() + "/30"
	connectionContext["dst_ip"] = dstIP.String() + "/30"

	// TODO take into consideration LocalMechnism preferences sent in request

	connection := &connection.Connection{
		Id:             request.GetConnection().GetId(),
		NetworkService: request.GetConnection().GetNetworkService(),
		Mechanism:      mechanism,
		Context:        connectionContext,
	}
	err = connection.IsComplete()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return connection, nil
}
