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
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/local"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

// MonitorEndpoint is a monitoring composite
type MonitorEndpoint struct {
	BaseCompositeEndpoint
	monitorConnectionServer local.MonitorServer
}

// Init will be called upon NSM Endpoint instantiation with the proper context
func (mce *MonitorEndpoint) Init(context *InitContext) error {
	grpcServer := context.GrpcServer
	connection.RegisterMonitorConnectionServer(grpcServer, mce.monitorConnectionServer)
	return nil
}

// Request implements the request handler
func (mce *MonitorEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.NetworkServiceReply, error) {

	if mce.GetNext() == nil {
		err := fmt.Errorf("Monitor needs next")
		logrus.Errorf("%v", err)
		return nil, err
	}

	reply, err := mce.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	logrus.Infof("Monitor UpdateConnection: %v", reply.GetConnection())
	mce.monitorConnectionServer.Update(reply.GetConnection())

	return reply, nil
}

// Close implements the close handler
func (mce *MonitorEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	logrus.Infof("Monitor DeleteConnection: %v", connection)
	mce.monitorConnectionServer.Delete(connection)
	if mce.GetNext() != nil {
		return mce.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (mce *MonitorEndpoint) Name() string {
	return "monitor"
}

// GetOpaque will return the monitor server
func (mce *MonitorEndpoint) GetOpaque(incoming interface{}) interface{} {
	return mce.monitorConnectionServer
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
