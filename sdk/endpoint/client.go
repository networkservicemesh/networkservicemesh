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

package endpoint

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

// ClientEndpoint - opens a Client connection to another Network Service
type ClientEndpoint struct {
	nsmClient     *client.NsmClient
	mechanismType string
	ioConnMap     map[string]*connection.Connection
}

// Request implements the request handler
// Consumes from ctx context.Context:
//	   Next
func (cce *ClientEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	name := request.GetConnection().GetId()
	outgoingConnection, err := cce.nsmClient.Connect(name, cce.mechanismType, "Describe "+name)
	if err != nil {
		logrus.Errorf("Error when creating the connection %v", err)
		return nil, err
	}
	ctx = WithClientConnection(ctx, outgoingConnection)
	incomingConnection := request.GetConnection()
	if Next(ctx) != nil {
		incomingConnection, err = Next(ctx).Request(ctx, request)
		if err != nil {
			return nil, err
		}
	}

	cce.ioConnMap[request.GetConnection().GetId()] = outgoingConnection
	logrus.Infof("outgoingConnection: %v", outgoingConnection)

	return incomingConnection, nil
}

// Close implements the close handler
// Consumes from ctx context.Context:
//	   Next
func (cce *ClientEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if outgoingConnection, ok := cce.ioConnMap[connection.GetId()]; ok {
		cce.nsmClient.Close(outgoingConnection)
	}
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (cce *ClientEndpoint) Name() string {
	return "client"
}

// NewClientEndpoint creates a ClientEndpoint
func NewClientEndpoint(configuration *common.NSConfiguration) *ClientEndpoint {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	nsmClient, err := client.NewNSMClient(context.Background(), configuration)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client %v", err)
		return nil
	}

	self := &ClientEndpoint{
		ioConnMap:     map[string]*connection.Connection{},
		mechanismType: configuration.MechanismType,
		nsmClient:     nsmClient,
	}

	return self
}
