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

package nscomposer

import (
	"encoding/binary"
	"net"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/sirupsen/logrus"
)

func (nsme *nsmEndpoint) CompleteConnection(request *networkservice.NetworkServiceRequest, outgoingConnection *connection.Connection) (*connection.Connection, error) {
	err := request.IsValid()
	if err != nil {
		return nil, err
	}

	if outgoingConnection == nil {
		outgoingConnection = &connection.Connection{
			Context: &connectioncontext.ConnectionContext{
				SrcIpRequired: true,
				DstIpRequired: true,
			},
		}
	}

	mechanism, err := connection.NewMechanism(nsme.mechanismType, "nsm"+nsme.id.MustGenerate(), "NSM Endpoint")
	if err != nil {
		return nil, err
	}

	// Force the connection src/dst IPs if configured
	if nsme.nextIP > 0 {
		srcIP := make(net.IP, 4)
		binary.BigEndian.PutUint32(srcIP, nsme.nextIP)
		nsme.nextIP = nsme.nextIP + 1

		dstIP := make(net.IP, 4)
		binary.BigEndian.PutUint32(dstIP, nsme.nextIP)
		nsme.nextIP = nsme.nextIP + 3

		outgoingConnection.Context = &connectioncontext.ConnectionContext{
			SrcIpAddr: srcIP.String() + "/30",
			DstIpAddr: dstIP.String() + "/30",
			Routes: []*connectioncontext.Route{
				{
					Prefix: "8.8.8.8/30",
				},
			},
		}
	}

	connection := &connection.Connection{
		Id:             request.GetConnection().GetId(),
		NetworkService: request.GetConnection().GetNetworkService(),
		Mechanism:      mechanism,
		Context:        outgoingConnection.GetContext(),
	}
	err = connection.IsComplete()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return connection, nil
}
