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
	"math/rand"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"github.com/teris-io/shortid"
)

type nsmEndpoint struct {
	*nsmConnection
	mechanismType           connection.MechanismType
	nextIP                  uint32
	ioConnMap               map[*connection.Connection]*nsmClient
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
	backend                 EndpointBackend
	id                      *shortid.Shortid
}

func (ns *nsmEndpoint) outgoingConnectionRequest(ctx context.Context, request *networkservice.NetworkServiceRequest) (*nsmClient, error) {
	client, err := NewNSMClient(ctx, ns.configuration, &dummyClientBackend{})
	if err != nil {
		logrus.Errorf("Unable to create the NSM client %v", err)
		return nil, err
	}

	client.name = client.name + request.GetConnection().GetId()
	client.mechanismType = request.GetMechanismPreferences()[0].GetType()
	client.Connect()

	// TODO: check this. Hack??
	client.GetConnection().GetMechanism().GetParameters()[connection.Workspace] = ""

	return client, nil
}

func (nsme *nsmEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Request for Network Service received %v", request)

	var client *nsmClient
	var err error
	if len(nsme.configuration.OutgoingNscName) > 0 {
		client, err = nsme.outgoingConnectionRequest(ctx, request)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
	}
	outgoingConnection := client.GetConnection()
	logrus.Infof("outgoingConnection: %v", outgoingConnection)

	incomingConnection, err := nsme.CompleteConnection(request, outgoingConnection)
	logrus.Infof("Completed incomingConnection %v", incomingConnection)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	err = nsme.backend.Request(ctx, incomingConnection, outgoingConnection, nsme.configuration.workspace)
	if err != nil {
		logrus.Errorf("The backend returned and error: %v", err)
		return nil, err
	}

	nsme.ioConnMap[incomingConnection] = client
	nsme.monitorConnectionServer.UpdateConnection(incomingConnection)
	logrus.Infof("Responding to NetworkService.Request(%v): %v", request, incomingConnection)
	return incomingConnection, nil
}

func (nsme *nsmEndpoint) Close(ctx context.Context, incomingConnection *connection.Connection) (*empty.Empty, error) {
	if outgoingConnection, ok := nsme.ioConnMap[incomingConnection]; ok {
		nsme.nsClient.Close(ctx, outgoingConnection.GetConnection())
	}
	nsme.backend.Close(ctx, incomingConnection, nsme.configuration.workspace)
	nsme.nsClient.Close(ctx, incomingConnection)
	nsme.monitorConnectionServer.DeleteConnection(incomingConnection)
	return &empty.Empty{}, nil
}

func (ns *nsmEndpoint) setupNSEServerConnection() (net.Listener, error) {
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

// NewNSMEndpoint creates a new NSM endpoint
func NewNSMEndpoint(ctx context.Context, configuration *NSConfiguration, backend EndpointBackend) (*nsmEndpoint, error) {
	if configuration == nil {
		configuration = &NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	if backend == nil {
		backend = &dummyEndpointBackend{}
	}

	nsmConnection, err := newNSMConnection(ctx, configuration)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		return nil, err
	}

	rand.Seed(time.Now().UTC().UnixNano())

	endpoint := &nsmEndpoint{
		nsmConnection:           nsmConnection,
		nextIP:                  ip2int(net.ParseIP(configuration.IPAddress)),
		ioConnMap:               map[*connection.Connection]*nsmClient{},
		monitorConnectionServer: monitor_connection_server.NewMonitorConnectionServer(),
		mechanismType:           mechanismFromString(configuration.OutgoingNscMechanism),
		backend:                 backend,
		id:                      shortid.MustNew(1, shortid.DEFAULT_ABC, rand.Uint64()),
	}

	err = endpoint.backend.New()
	if err != nil {
		logrus.Errorf("Unable to create the endpoint backend. Error: %v", err)
		return nil, err
	}

	return endpoint, nil
}
