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
package remote

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/remote"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
)

type monitorService struct {
	monitor remote.MonitorServer
}

// NewMonitorService - Perform updates to workspace monitoring services.
func NewMonitorService(monitor remote.MonitorServer) networkservice.NetworkServiceServer {
	return &monitorService{
		monitor: monitor,
	}
}

func (srv *monitorService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = common.WithRemoteMonitorServer(ctx, srv.monitor)

	conn, err := ProcessNext(ctx, request)
	if conn != nil {
		srv.monitor.Update(conn)
	}
	return conn, err
}

func (srv *monitorService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	logrus.Infof("Closing connection: %v", connection)

	// Pass model connection with context
	ctx = common.WithRemoteMonitorServer(ctx, srv.monitor)
	conn, err := ProcessClose(ctx, connection)
	srv.monitor.Delete(connection)
	return conn, err
}
