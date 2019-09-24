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
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/local"
)

// MonitorEndpoint is a monitoring composite
type MonitorEndpoint struct {
	monitorConnectionServer local.MonitorServer
}

// Init will be called upon NSM Endpoint instantiation with the proper context
func (mce *MonitorEndpoint) Init(context *InitContext) error {
	grpcServer := context.GrpcServer
	connection.RegisterMonitorConnectionServer(grpcServer, mce.monitorConnectionServer)
	return nil
}

// Request implements the request handler
// Consumes from ctx context.Context:
//     MonitorServer
//	   Next
func (mce *MonitorEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if Next(ctx) != nil {

		// Pass monitor server
		ctx = WithMonitorServer(ctx, mce.monitorConnectionServer)

		incomingConnection, err := Next(ctx).Request(ctx, request)
		if err != nil {
			Log(ctx).Errorf("Next request failed: %v", err)
			return nil, err
		}

		Log(ctx).Infof("Monitor UpdateConnection: %v", incomingConnection)
		mce.monitorConnectionServer.Update(incomingConnection)

		return incomingConnection, nil
	}
	return nil, fmt.Errorf("MonitorEndpoint.Request - cannot create requested connection")
}

// Close implements the close handler
// Request implements the request handler
// Consumes from ctx context.Context:
//     MonitorServer
//	   Next
func (mce *MonitorEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	Log(ctx).Infof("Monitor DeleteConnection: %v", connection)

	// Pass monitor server
	ctx = WithMonitorServer(ctx, mce.monitorConnectionServer)

	if Next(ctx) != nil {
		rv, err := Next(ctx).Close(ctx, connection)
		mce.monitorConnectionServer.Delete(connection)
		return rv, err
	}
	return nil, fmt.Errorf("monitor DeleteConnection cannot close connection")
}

// Name returns the composite name
func (mce *MonitorEndpoint) Name() string {
	return "monitor"
}

// NewMonitorEndpoint creates a MonitorEndpoint
func NewMonitorEndpoint(configuration *common.NSConfiguration) *MonitorEndpoint {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	self := &MonitorEndpoint{
		monitorConnectionServer: local.NewMonitorServer(),
	}

	return self
}
