// Copyright (c) 2019 Cisco and/or its affiliates.
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

package local

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
)

// ConnectionService makes basic Mechanism selection for the incoming connection
type сrossConnectService struct {
}

func (cce *сrossConnectService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logger := common.Log(ctx)
	endpointConnection := common.EndpointConnection(ctx)
	endpoint := common.Endpoint(ctx)
	clientConnection := common.ModelConnection(ctx)

	if endpointConnection == nil || endpoint == nil || clientConnection == nil {
		err := fmt.Errorf("endpoint connection/Endpoint/ClientConnection should be specified with context")
		logger.Error(err)
		return nil, err
	}

	// 7.2.6.2.4 create cross connection
	dpAPIConnection := crossconnect.NewCrossConnect(
		request.Connection.GetId(),
		endpoint.GetNetworkService().GetPayload(),
		request.Connection,
		endpointConnection,
	)
	clientConnection.Xcon = dpAPIConnection

	return ProcessNext(ctx, request)
}

func (cce *сrossConnectService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	return ProcessClose(ctx, connection)
}

// NewCrossConnectService -  creates a service to create a cross connect
func NewCrossConnectService() networkservice.NetworkServiceServer {
	return &сrossConnectService{}
}
