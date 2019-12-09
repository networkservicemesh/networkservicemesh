// Copyright (c) 2019 Cisco and/or its affiliates.
//
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

package common

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

type RequestBuilder interface {
	Build(*registry.NSERegistration, *connection.Connection, *model.ClientConnection) *networkservice.NetworkServiceRequest
}

type LocalNSERequestBuilder struct {
	localMechanisms []*connection.Mechanism
	nsmName         string
	connectionId    string
}

func (b *LocalNSERequestBuilder) Build(endpoint *registry.NSERegistration, requestConn *connection.Connection, clientConnection *model.ClientConnection) *networkservice.NetworkServiceRequest {
	// We need to obtain parameters for local mechanism
	localM := append([]*connection.Mechanism{}, b.localMechanisms...)

	if clientConnection.ConnectionState == model.ClientConnectionHealing && endpoint == clientConnection.Endpoint {
		if localDst := clientConnection.Xcon.GetLocalDestination(); localDst != nil {
			return &networkservice.NetworkServiceRequest{
				Connection: &connection.Connection{
					Id:                     localDst.GetId(),
					NetworkService:         localDst.NetworkService,
					Context:                localDst.GetContext(),
					Labels:                 localDst.GetLabels(),
					NetworkServiceManagers: []string{b.nsmName},
				},
				MechanismPreferences: localM,
			}
		}
	}

	return &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id:                     b.connectionId, // ID for NSE is managed by NSMgr
			NetworkService:         endpoint.GetNetworkService().GetName(),
			NetworkServiceManagers: []string{b.nsmName},
			Context:                requestConn.GetContext(),
			Labels:                 requestConn.GetLabels(),
		},
		MechanismPreferences: localM,
	}
}

type RemoteNSMRequestBuilder struct {
	remoteMechanisms []*connection.Mechanism
	srcNsmName       string
	dstNsmName       string
}

func (b *RemoteNSMRequestBuilder) Build(endpoint *registry.NSERegistration,
	requestConn *connection.Connection, clientConnection *model.ClientConnection) *networkservice.NetworkServiceRequest {
	// We need to obtain parameters for remote mechanism
	remoteM := append([]*connection.Mechanism{}, b.remoteMechanisms...)

	// Try Heal only if endpoint are same as for existing connection.
	if clientConnection.ConnectionState == model.ClientConnectionHealing && endpoint == clientConnection.Endpoint {
		if remoteDst := clientConnection.Xcon.GetRemoteDestination(); remoteDst != nil {
			return &networkservice.NetworkServiceRequest{
				Connection: &connection.Connection{
					Id:                         remoteDst.GetId(),
					NetworkService:             remoteDst.NetworkService,
					Context:                    remoteDst.GetContext(),
					Labels:                     remoteDst.GetLabels(),
					NetworkServiceEndpointName: endpoint.GetNetworkServiceEndpoint().GetName(),
					NetworkServiceManagers: []string{
						b.srcNsmName, // src
						b.dstNsmName, // dst
					},
				},
				MechanismPreferences: remoteM,
			}
		}
	}

	return &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id:                         "-",
			NetworkService:             requestConn.GetNetworkService(),
			Context:                    requestConn.GetContext(),
			Labels:                     requestConn.GetLabels(),
			NetworkServiceEndpointName: endpoint.GetNetworkServiceEndpoint().GetName(),
			NetworkServiceManagers: []string{
				b.srcNsmName, // src
				b.dstNsmName, // dst
			},
		},
		MechanismPreferences: remoteM,
	}
}
