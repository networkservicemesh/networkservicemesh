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

// Package connectionmonitor - implementation of connection monotor client and server
package connectionmonitor

import (
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
)

// MonitorServer is a monitor.Server for local/connection GRPC API
type MonitorServer interface {
	monitor.Server
	connection.MonitorConnectionServer
}

type monitorServer struct {
	monitor.Server
	factoryName string
}

// NewMonitorServer creates a new MonitorServer
func NewMonitorServer(factoryName string) MonitorServer {
	rv := &monitorServer{
		factoryName: factoryName,
		Server: monitor.NewServer(&eventFactory{
			factoryName: factoryName,
		}),
	}
	go rv.Serve()
	return rv
}

// MonitorConnections adds recipient for MonitorServer events
func (s *monitorServer) MonitorConnections(in *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	if in != nil {
		logrus.Infof("%sMonitor using filter %v", s.factoryName, in)
		recipient = NewMonitorConnectionFilter(in, recipient)
	}
	s.MonitorEntities(recipient)
	return nil
}
