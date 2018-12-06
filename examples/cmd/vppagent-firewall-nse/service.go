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
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/sirupsen/logrus"
)

type vppagentNetworkService struct {
	sync.RWMutex
	outgoingNscName         string
	outgoingNscLabels       map[string]string
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
	vppAgentEndpoint        string
	baseDir                 string
	clientConnection        networkservice.NetworkServiceClient
	crossConnects           map[string]*crossconnect.CrossConnect
}

func (ns *vppagentNetworkService) outgoingConnectionRequest(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Initiating an outgoing connection.")

	outgoingMechanism, err := connection.NewMechanism(request.GetMechanismPreferences()[0].GetType(),
		"firewall", "A firewall outgoing interface")
	if err != nil {
		logrus.Errorf("Failure to prepare the outgoing mechanism preference with error: %+v", err)
		return nil, err
	}

	outgoingRequest := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: ns.outgoingNscName,
			Context: map[string]string{
				"requires": "src_ip,dst_ip",
			},
			Labels: ns.outgoingNscLabels,
		},
		MechanismPreferences: []*connection.Mechanism{
			outgoingMechanism,
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

	return outgoingConnection, nil
}

func (ns *vppagentNetworkService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Request for Network Service received %v", request)

	outgoingConnection, err := ns.outgoingConnectionRequest(ctx, request)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	outgoingConnection.GetMechanism().GetParameters()[connection.Workspace] = ""
	logrus.Infof("outgoingConnection: %v", outgoingConnection)

	incomingConnection, err := ns.CompleteConnection(request, outgoingConnection)
	logrus.Infof("Completed incomingConnection %v", incomingConnection)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	crossConnectRequest := &crossconnect.CrossConnect{
		Id:      request.GetConnection().GetId(),
		Payload: "IP",
		Source: &crossconnect.CrossConnect_LocalSource{
			incomingConnection,
		},
		Destination: &crossconnect.CrossConnect_LocalDestination{
			outgoingConnection,
		},
	}

	crossConnect, dataChange, err := ns.CrossConnecVppInterfaces(ctx, crossConnectRequest, true, ns.baseDir)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	// The Crossconnect converter generates and puts the Source Interface name here
	ingressIfName := dataChange.XCons[0].ReceiveInterface

	aclRules := map[string]string{
		"Allow ICMP":           "action=permit,icmptype=8",
		"Deny everything else": "action=deny",
	}

	err = ns.ApplyAclOnVppInterface(ctx, "IngressACL", ingressIfName, aclRules)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	// Store for cleanup
	ns.crossConnects[incomingConnection.GetId()] = crossConnect

	ns.monitorConnectionServer.UpdateConnection(incomingConnection)
	logrus.Infof("Responding to NetworkService.Request(%v): %v", request, incomingConnection)
	return incomingConnection, nil
}

func (ns *vppagentNetworkService) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	// remove from connection
	crossConnectRequest, ok := ns.crossConnects[conn.GetId()]
	if ok {
		_, _, err := ns.CrossConnecVppInterfaces(ctx, crossConnectRequest, false, ns.baseDir)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
	}

	ns.monitorConnectionServer.DeleteConnection(conn)
	return &empty.Empty{}, nil
}

func New(outgoingNscName, vppAgentEndpoint string, baseDir string, outgoingNscLabels map[string]string, clientConnection networkservice.NetworkServiceClient) networkservice.NetworkServiceServer {
	monitor := monitor_connection_server.NewMonitorConnectionServer()
	service := vppagentNetworkService{
		outgoingNscName:         outgoingNscName,
		outgoingNscLabels:       outgoingNscLabels,
		monitorConnectionServer: monitor,
		vppAgentEndpoint:        vppAgentEndpoint,
		baseDir:                 baseDir,
		clientConnection:        clientConnection,
		crossConnects:           make(map[string]*crossconnect.CrossConnect),
	}
	service.Reset()
	return &service
}
