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
	"encoding/binary"
	"net"
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
	nextIP                  uint32
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
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
	// remove from connection
	ns.monitorConnectionServer.DeleteConnection(conn)
	return &empty.Empty{}, nil
}

func ip2int(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func New(ip string) networkservice.NetworkServiceServer {
	monitor := monitor_connection_server.NewMonitorConnectionServer()
	netIP := net.ParseIP(ip)
	service := networkService{
		networkService:          "icmp-responder",
		nextIP:                  ip2int(netIP), // 10.20.1.1
		monitorConnectionServer: monitor,
	}
	return &service
}
