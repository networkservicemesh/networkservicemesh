// Copyright (c) 2019 Cisco Systems, Inc and/or its affiliates.
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

package connectionmonitor

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
)

type eventStream struct {
	connection.MonitorConnection_MonitorConnectionsClient
}

func (s *eventStream) Recv() (interface{}, error) {
	return s.MonitorConnection_MonitorConnectionsClient.Recv()
}

// NewMonitorClient creates a new monitor.Client for local/connection GRPC API
func NewMonitorClient(cc *grpc.ClientConn, in *connection.MonitorScopeSelector) (monitor.Client, error) {
	newEventStream := func(ctx context.Context, cc *grpc.ClientConn) (monitor.EventStream, error) {
		stream, err := connection.NewMonitorConnectionClient(cc).MonitorConnections(ctx, in)

		return &eventStream{
			MonitorConnection_MonitorConnectionsClient: stream,
		}, err
	}
	return monitor.NewClient(cc, &eventFactory{}, newEventStream)
}
