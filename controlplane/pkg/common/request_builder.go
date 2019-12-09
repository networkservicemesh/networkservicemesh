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
	Build(string, *registry.NSERegistration, *model.Forwarder, *connection.Connection) *networkservice.NetworkServiceRequest
}

type LocalNSERequestBuilder struct {
	nsmName     string
	idGenerator func() string
}

func (builder *LocalNSERequestBuilder) Build(connectionId string, endpoint *registry.NSERegistration, fwd *model.Forwarder, requestConn *connection.Connection) *networkservice.NetworkServiceRequest {
	if connectionId == "" {
		connectionId = builder.idGenerator()
	}
	return &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id:                     connectionId, // ID for NSE is managed by NSMgr
			NetworkService:         requestConn.GetNetworkService(),
			Context:                requestConn.GetContext(),
			Labels:                 requestConn.GetLabels(),
			NetworkServiceManagers: []string{builder.nsmName},
		},
		MechanismPreferences: fwd.LocalMechanisms,
	}
}

type RemoteNSMRequestBuilder struct {
	srcNsmName string
}

func (builder *RemoteNSMRequestBuilder) Build(connectionId string, endpoint *registry.NSERegistration, fwd *model.Forwarder,
	requestConn *connection.Connection) *networkservice.NetworkServiceRequest {

	return &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id:                         connectionId,
			NetworkService:             requestConn.GetNetworkService(),
			Context:                    requestConn.GetContext(),
			Labels:                     requestConn.GetLabels(),
			NetworkServiceEndpointName: endpoint.GetNetworkServiceEndpoint().GetName(),
			NetworkServiceManagers: []string{
				builder.srcNsmName, // src
				endpoint.GetNetworkServiceManager().GetName(), // dst
			},
		},
		MechanismPreferences: fwd.RemoteMechanisms,
	}
}
