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

package nscomposer

import (
	"context"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
)

type nsEndpoint struct {
	sync.RWMutex
	configuration           *NSConfiguration
	outgoingNscName         string
	outgoingNscLabels       map[string]string
	workspace               string
	nextIP                  uint32
	ioConnMap               map[*connection.Connection]*nsmClient
	clientConnection        networkservice.NetworkServiceClient
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
	backend                 nsComposerBackend
}

type outgoingClientBackend struct{}

func (ocb *outgoingClientBackend) New() error { return nil }
func (ocb *outgoingClientBackend) Connect(ctx context.Context, connection *connection.Connection) error {
	return nil
}
func (ocb *outgoingClientBackend) Close(ctx context.Context, connection *connection.Connection) error {
	return nil
}

func (ns *nsEndpoint) outgoingConnectionRequest(ctx context.Context, request *networkservice.NetworkServiceRequest) (*nsmClient, error) {
	client, err := NewNSMClient(ctx, ns.configuration, &outgoingClientBackend{})
	if err != nil {
		logrus.Errorf("Unable to create the NSM client %v", err)
		return nil, err
	}

	client.name = client.name + request.GetConnection().GetId()
	client.mechanismType = request.GetMechanismPreferences()[0].GetType()
	client.Connect()

	// Hack??
	client.GetConnection().GetMechanism().GetParameters()[connection.Workspace] = ""

	return client, nil
}

func (ns *nsEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Request for Network Service received %v", request)

	var client *nsmClient
	var err error
	if len(ns.outgoingNscName) > 0 {
		client, err = ns.outgoingConnectionRequest(ctx, request)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
	}
	outgoingConnection := client.GetConnection()
	logrus.Infof("outgoingConnection: %v", outgoingConnection)

	incomingConnection, err := ns.CompleteConnection(request, outgoingConnection)
	logrus.Infof("Completed incomingConnection %v", incomingConnection)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	err = ns.backend.Request(ctx, incomingConnection, outgoingConnection, ns.workspace)
	if err != nil {
		logrus.Errorf("The backend returned and error: %v", err)
		return nil, err
	}

	ns.ioConnMap[incomingConnection] = client
	ns.monitorConnectionServer.UpdateConnection(incomingConnection)
	logrus.Infof("Responding to NetworkService.Request(%v): %v", request, incomingConnection)
	return incomingConnection, nil
}

func (ns *nsEndpoint) Close(ctx context.Context, incomingConnection *connection.Connection) (*empty.Empty, error) {
	if outgoingConnection, ok := ns.ioConnMap[incomingConnection]; ok {
		ns.clientConnection.Close(ctx, outgoingConnection.GetConnection())
	}
	ns.clientConnection.Close(ctx, incomingConnection)
	ns.backend.Close(ctx, incomingConnection, ns.workspace)
	ns.monitorConnectionServer.DeleteConnection(incomingConnection)
	return &empty.Empty{}, nil
}

func (ns *nsEndpoint) setupNSEServerConnection() (net.Listener, error) {
	c := ns.configuration
	if err := tools.SocketCleanup(c.nsmClientSocket); err != nil {
		logrus.Errorf("nse: failure to cleanup stale socket %s with error: %v", c.nsmClientSocket, err)
		return nil, err
	}

	logrus.Infof("nse: listening socket %s", c.nsmClientSocket)
	connectionServer, err := net.Listen("unix", c.nsmClientSocket)
	if err != nil {
		logrus.Errorf("nse: failure to listen on a socket %s with error: %v", c.nsmClientSocket, err)
		return nil, err
	}
	return connectionServer, nil
}

func newNsEndpoint(configuration *NSConfiguration, clientConnection networkservice.NetworkServiceClient, backend nsComposerBackend) (*nsEndpoint, error) {
	err := backend.New()
	if err != nil {
		logrus.Errorf("Unable to create the backend. Error: %v", err)
		return nil, err
	}
	netIP := net.ParseIP(configuration.IPAddress)
	return &nsEndpoint{
		configuration:           configuration,
		outgoingNscName:         configuration.OutgoingNscName,
		outgoingNscLabels:       tools.ParseKVStringToMap(configuration.OutgoingNscLabels, ",", "="),
		workspace:               configuration.workspace,
		nextIP:                  ip2int(netIP),
		ioConnMap:               map[*connection.Connection]*nsmClient{},
		clientConnection:        clientConnection,
		monitorConnectionServer: monitor_connection_server.NewMonitorConnectionServer(),
		backend:                 backend,
	}, nil
}
