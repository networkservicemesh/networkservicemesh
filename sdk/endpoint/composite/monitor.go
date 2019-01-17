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
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/ligato/networkservicemesh/sdk/common"
	"github.com/ligato/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
)

// MonitorCompositeEndpoint is a monitoring composite
type MonitorCompositeEndpoint struct {
	endpoint.BaseCompositeEndpoint
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
}

// Request imeplements the request handler
func (mce *MonitorCompositeEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if mce.GetNext() == nil {
		err := fmt.Errorf("Monitor needs next")
		logrus.Errorf("%v", err)
		return nil, err
	}

	incomingConnection, err := mce.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	mce.monitorConnectionServer.UpdateConnection(incomingConnection)

	return incomingConnection, nil
}

// Close imeplements the close handler
func (mce *MonitorCompositeEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if mce.GetNext() != nil {
		return mce.GetNext().Close(ctx, connection)
	}
	mce.monitorConnectionServer.DeleteConnection(connection)
	return &empty.Empty{}, nil
}

// NewMonitorCompositeEndpoint creates a MonitorCompositeEndpoint
func NewMonitorCompositeEndpoint(configuration *common.NSConfiguration) *MonitorCompositeEndpoint {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	self := &MonitorCompositeEndpoint{
		monitorConnectionServer: monitor_connection_server.NewMonitorConnectionServer(),
	}
	self.SetSelf(self)

	return self
}
