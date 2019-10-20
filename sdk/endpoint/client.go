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

	"github.com/hashicorp/go-multierror"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

// ClientEndpoint - opens a Client connection to another Network Service
type ClientEndpoint struct {
	mechanismType string
	ioConnMap     map[string]*connection.Connection
	configuration *common.NSConfiguration
}

// Request implements the request handler
// Consumes from ctx context.Context:
//	   Next
func (cce *ClientEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	name := request.GetConnection().GetId()

	nsmClient, err := client.NewNSMClient(ctx, cce.configuration)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client %v", err)
		return nil, err
	}
	defer func() {
		if deleteErr := nsmClient.Destroy(ctx); deleteErr != nil {
			logrus.Errorf("error destroying nsm client %v", deleteErr)
		}
	}()

	outgoingConnection, err := nsmClient.Connect(ctx, name, cce.mechanismType, "Describe "+name)
	if err != nil {
		logrus.Errorf("Error when creating the connection %v", err)
		return nil, err
	}

	//TODO: Do we need this ?
	outgoingConnection.GetMechanism().GetParameters()[connection.Workspace] = ""
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
	var result error

	nsmClient, err := client.NewNSMClient(ctx, cce.configuration)
	if err != nil {
		logrus.Fatalf("Unable to create the NSM client %v", err)
		return nil, err
	}
	defer func() {
		if err := nsmClient.Destroy(ctx); err != nil {
			logrus.Errorf("error destroy nsm client %v", err)
		}
	}()
	if outgoingConnection, ok := cce.ioConnMap[connection.GetId()]; ok {
		if err := nsmClient.Close(ctx, outgoingConnection); err != nil {
			result = multierror.Append(result, err)
		}
		ctx = WithClientConnection(ctx, outgoingConnection)
	}
	// Remove collection from map, after all child items are passed
	defer delete(cce.ioConnMap, connection.GetId())
	if Next(ctx) != nil {
		if _, err := Next(ctx).Close(ctx, connection); err != nil {
			return &empty.Empty{}, multierror.Append(result, err)
		}
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

	self := &ClientEndpoint{
		ioConnMap:     map[string]*connection.Connection{},
		mechanismType: configuration.MechanismType,
		configuration: configuration,
	}

	return self
}
