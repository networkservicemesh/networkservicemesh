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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/ligato/vpp-agent/plugins/vpp/model/l2"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"sync"
)

type vppagentNetworkService struct {
	sync.RWMutex
	networkService          string
	nextIP                  uint32
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
	vppAgentEndpoint        string
	baseDir                 string
	state                   *l2.BridgeDomains_BridgeDomain
	splitHorizonGroup       int
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

func ip2int(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func New(vppAgentEndpoint, baseDir, ip, bridgeDomainName string) networkservice.NetworkServiceServer {
	monitor := monitor_connection_server.NewMonitorConnectionServer()
	netIP := net.ParseIP(ip)

	shg := 0
	for shg == 0 {
		shg = rand.Int()
	}

	bridgeDomain := &l2.BridgeDomains_BridgeDomain{
		Name:                 bridgeDomainName,
		Flood:                true,
		UnknownUnicastFlood:  true,
		Forward:              true,
		Learn:                true,
		ArpTermination:       false,
		MacAge:               120,
		Interfaces:           nil,
		ArpTerminationTable:  nil,
	}

	service := vppagentNetworkService{
		networkService:          NetworkServiceName,
		nextIP:                  ip2int(netIP),
		monitorConnectionServer: monitor,
		vppAgentEndpoint:        vppAgentEndpoint,
		baseDir:                 baseDir,
		state:                   bridgeDomain,
		splitHorizonGroup: 1,
	}

	service.Reset()
	service.CreateBridgeDomain(context.Background())

	return &service
}
