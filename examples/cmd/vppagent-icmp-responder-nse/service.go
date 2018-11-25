// Copyright (c) 2018 Cisco and/or its affiliates.
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

package main

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/sirupsen/logrus"
)

type vppagentNetworkService struct {
	sync.RWMutex
	networkService          string
	nextIP                  uint32
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
	vppAgentEndpoint        string
	baseDir                 string
}

func (ns *vppagentNetworkService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Request for Network Service received %v", request)
	nseConnection, err := ns.CompleteConnection(request)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	if err := ns.CreateVppInterface(ctx, nseConnection, ns.baseDir); err != nil {
		return nil, err
	}

	ns.monitorConnectionServer.UpdateConnection(nseConnection)
	logrus.Infof("Responding to NetworkService.Request(%v): %v", request, nseConnection)
	return nseConnection, nil
}

func (ns *vppagentNetworkService) Close(_ context.Context, conn *connection.Connection) (*empty.Empty, error) {
	// remove from connection
	ns.monitorConnectionServer.DeleteConnection(conn)
	return &empty.Empty{}, nil
}

func New(vppAgentEndpoint string, baseDir string) networkservice.NetworkServiceServer {
	monitor := monitor_connection_server.NewMonitorConnectionServer()
	service := vppagentNetworkService{
		networkService:          NetworkServiceName,
		nextIP:                  169738497, // 10.30.1.1
		monitorConnectionServer: monitor,
		vppAgentEndpoint:        vppAgentEndpoint,
		baseDir:                 baseDir,
	}
	service.Reset()
	return &service
}
