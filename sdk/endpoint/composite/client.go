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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
)

type ClientCompositeEndpoint struct {
	endpoint.BaseCompositeEndpoint
	nsmClient     *client.NsmClient
	mechanismType string
	ioConnMap     map[string]*connection.Connection
}

// Request imeplements the request handler
func (cce *ClientCompositeEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if cce.GetNext() == nil {
		logrus.Fatal("The connection composite requires that there is Next set.")
	}

	incomingConnection, err := cce.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	var outgoingConnection *connection.Connection
	name := request.GetConnection().GetId()
	outgoingConnection, err = cce.nsmClient.Connect(name, cce.mechanismType, "Describe "+name)
	if err != nil {
		logrus.Errorf("Error when creating the connection %v", err)
		return nil, err
	}

	// TODO: check this. Hack??
	outgoingConnection.GetMechanism().GetParameters()[connection.Workspace] = ""

	cce.ioConnMap[incomingConnection.GetId()] = outgoingConnection
	logrus.Infof("outgoingConnection: %v", outgoingConnection)

	return incomingConnection, nil
}

// Close imeplements the close handler
func (cce *ClientCompositeEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if outgoingConnection, ok := cce.ioConnMap[connection.GetId()]; ok {
		cce.nsmClient.Close(outgoingConnection)
	}
	if cce.GetNext() != nil {
		return cce.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// GetOpaque will return the corresponding outgoing connection
func (cce *ClientCompositeEndpoint) GetOpaque(incoming interface{}) interface{} {
	incomingConnection := incoming.(*connection.Connection)
	if outgoingConnection, ok := cce.ioConnMap[incomingConnection.GetId()]; ok {
		return outgoingConnection
	}
	logrus.Errorf("GetOpaque outgoing not found for %v", incomingConnection)
	return nil
}

// NewClientCompositeEndpoint creates a ClientCompositeEndpoint
func NewClientCompositeEndpoint(configuration *common.NSConfiguration) *ClientCompositeEndpoint {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	nsmClient, err := client.NewNSMClient(nil, configuration)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client %v", err)
		return nil
	}

	self := &ClientCompositeEndpoint{
		ioConnMap:     map[string]*connection.Connection{},
		mechanismType: configuration.MechanismType,
		nsmClient:     nsmClient,
	}
	self.SetSelf(self)

	return self
}
