// Copyright 2018 VMware, Inc.
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

package main

import (
	"context"
	"path"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/sirupsen/logrus"
)

type vppagentNetworkService struct {
	sync.RWMutex
	networkServiceName      string
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
	vppAgentEndpoint        string
	baseDir                 string
	clientConnection        networkservice.NetworkServiceClient
}

func (ns *vppagentNetworkService) outgoingConnectionRequest(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Initiating an outgoing connection.")

	outgoingRequest := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: ns.networkServiceName,
			Context: map[string]string{
				"requires": "src_ip,dst_ip",
			},
			Labels: map[string]string{
				"app": "firewall", // TODO - make these ENV configurable
			},
		},
		MechanismPreferences: []*connection.Mechanism{
			{
				Type: connection.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					connection.InterfaceNameKey: "firewall",
					connection.SocketFilename:   path.Join("firewall", "memif.sock"),
				},
			},
		},
	}

	var outgoingConnection *connection.Connection
	for ; true; <-time.After(5 * time.Second) {
		var err error
		logrus.Infof("Sending outgoing request %v", outgoingRequest)
		outgoingConnection, err = ns.clientConnection.Request(context.Background(), outgoingRequest)

		if err != nil {
			logrus.Errorf("failure to request connection with error: %+v", err)
			continue
		}
		logrus.Infof("Received outgoing connection: %v", outgoingConnection)
		break
	}

	// vppInterfaceConnection os the same as outgoingConnection minus the context
	vppInterfaceConnection := connection.Connection{
		Id:             outgoingConnection.GetId(),
		NetworkService: outgoingConnection.GetNetworkService(),
		Mechanism:      outgoingConnection.GetMechanism(),
		Context:        map[string]string{},
		Labels:         outgoingConnection.GetLabels(),
	}
	if err := ns.CreateVppInterfaceSrc(ctx, &vppInterfaceConnection, ns.baseDir); err != nil {
		logrus.Fatal(err)
	}

	return outgoingConnection, nil
}

func (ns *vppagentNetworkService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Request for Network Service received %v", request)

	outgoingConnection, err := ns.outgoingConnectionRequest(ctx, request)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	incomingConnection, err := ns.CompleteConnection(request, outgoingConnection)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	if err := ns.CreateVppInterfaceDst(ctx, incomingConnection, ns.baseDir); err != nil {
		return nil, err
	}

	ns.monitorConnectionServer.UpdateConnection(incomingConnection)
	logrus.Infof("Responding to NetworkService.Request(%v): %v", request, incomingConnection)
	return incomingConnection, nil
}

func (ns *vppagentNetworkService) Close(_ context.Context, conn *connection.Connection) (*empty.Empty, error) {
	// remove from connection
	ns.monitorConnectionServer.DeleteConnection(conn)
	return &empty.Empty{}, nil
}

func New(networkServiceName, vppAgentEndpoint string, baseDir string, clientConnection networkservice.NetworkServiceClient) networkservice.NetworkServiceServer {
	monitor := monitor_connection_server.NewMonitorConnectionServer()
	service := vppagentNetworkService{
		networkServiceName:      networkServiceName,
		monitorConnectionServer: monitor,
		vppAgentEndpoint:        vppAgentEndpoint,
		baseDir:                 baseDir,
		clientConnection:        clientConnection,
	}
	service.Reset()
	return &service
}
