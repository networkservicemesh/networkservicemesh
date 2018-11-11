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
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/networkservice"
	"github.com/ligato/networkservicemesh/dataplanes/vpp/pkg/nsmutils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"sync"
)

type networkService struct {
	sync.RWMutex
	networkService string
	nextIP         uint32
	requestChan    chan message
	connections    map[string]*networkservice.Connection
	monitors       map[int64]chan message
}

type message struct {
	message    string
	connection *networkservice.Connection
}

func (ns *networkService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logrus.Info("Request from nsm for connection: %v", request)
	linuxNS, _ := tools.GetCurrentNS()

	connectionContext := &networkservice.ConnectionContext{
		ConnectionContext: make(map[string]string),
	}

	srcIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(srcIP, ns.nextIP)
	ns.nextIP = ns.nextIP + 1

	dstIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(dstIP, ns.nextIP)
	ns.nextIP = ns.nextIP + 3

	connectionContext.ConnectionContext["src_ip"] = srcIP.String() + "/30"
	connectionContext.ConnectionContext["dst_ip"] = dstIP.String() + "/30"

	connection := &networkservice.Connection{
		ConnectionId:   request.Connection.ConnectionId,
		NetworkService: request.Connection.NetworkService,
		LocalMechanism: &common.LocalMechanism{
			Type: request.LocalMechanismPreference[0].Type,
			Parameters: map[string]string{
				nsmutils.NSMkeyNamespace:        linuxNS,
				nsmutils.NSMkeyIPv4:             "2.2.2.3",
				nsmutils.NSMkeyIPv4PrefixLength: "24",
			},
		},
		ConnectionContext: connectionContext,
	}

	ns.requestChan <- message{"created", connection}

	return connection, nil
}

func (ns *networkService) Close(_ context.Context, connection *networkservice.Connection) (*networkservice.Connection, error) {
	// remove from connection
	ns.requestChan <- message{"close", connection}
	return connection, nil
}

func (ns *networkService) Monitor(connection *networkservice.Connection, monitorServer networkservice.NetworkService_MonitorServer) error {
	monitor := make(chan message)
	key := rand.Int63()

	ns.Lock()
	ns.monitors[key] = monitor
	ns.Unlock()

	defer func() {
		ns.Lock()
		delete(ns.monitors, key)
		ns.Unlock()
	}()

	for msg := range monitor {
		if msg.connection.ConnectionId == connection.ConnectionId {
			err := monitorServer.Send(msg.connection)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ns *networkService) MonitorConnections(_ *common.Empty, monitorServer networkservice.NetworkService_MonitorConnectionsServer) error {
	monitor := make(chan message)
	key := rand.Int63()

	ns.Lock()
	ns.monitors[key] = monitor
	ns.Unlock()

	defer func() {
		ns.Lock()
		delete(ns.monitors, key)
		ns.Unlock()
	}()

	for msg := range monitor {
		err := monitorServer.Send(msg.connection)
		if err != nil {
			return err
		}
	}
	return nil
}

func New() networkservice.NetworkServiceServer {
	requestChan := make(chan message)
	service := networkService{
		networkService: "icmp-responder",
		nextIP:         169083137, // 10.20.1.1
		requestChan:    requestChan,
		connections:    make(map[string]*networkservice.Connection),
		monitors:       make(map[int64]chan message),
	}

	go func() {
		for nextMessage := range service.requestChan {
			service.RLock()
			for _, monitor := range service.monitors {
				monitor <- nextMessage
			}
			service.RUnlock()
		}
	}()

	return &service
}
