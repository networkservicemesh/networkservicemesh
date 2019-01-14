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
	"github.com/ligato/networkservicemesh/controlplane/pkg/prefix_pool"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/sirupsen/logrus"
)

type networkService struct {
	sync.RWMutex
	networkService          string
	prefixPool              prefix_pool.PrefixPool
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
	netNS string
}

type message struct {
	message    string
	connection *connection.Connection
}

func (ns *networkService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Request for Network Service received %v", request)
	conn, err := ns.CompleteConnection(request)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	ns.monitorConnectionServer.UpdateConnection(conn)
	return conn, nil
}

func (ns *networkService) Close(_ context.Context, conn *connection.Connection) (*empty.Empty, error) {
	logrus.Infof("Close for Network Service received %v", conn)
	// remove from connection
	ns.monitorConnectionServer.DeleteConnection(conn)
	prefix, requests, err := ns.prefixPool.GetConnectionInformation(conn.Id)
	logrus.Infof("Release connection prefixes network: %s extra requests: %v", prefix, requests)
	if err != nil {
		return &empty.Empty{}, err
	}
	err = ns.prefixPool.Release(conn.Id)
	return &empty.Empty{}, err
}

func New(ip string) networkservice.NetworkServiceServer {
	monitor := monitor_connection_server.NewMonitorConnectionServer()
	pool, err := prefix_pool.NewPrefixPool(ip)
	if err != nil {
		panic(err.Error())
	}
	netns, _ := tools.GetCurrentNS()

	service := networkService{
		networkService:          "icmp-responder",
		prefixPool:              pool, // 10.20.1.1
		monitorConnectionServer: monitor,
		netNS: netns,
	}
	return &service
}
