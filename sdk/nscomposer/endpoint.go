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

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"github.com/teris-io/shortid"
	"google.golang.org/grpc"
)

type nsmEndpoint struct {
	*nsmConnection
	mechanismType           connection.MechanismType
	nextIP                  uint32
	ioConnMap               map[*connection.Connection]*nsmClient
	monitorConnectionServer monitor_connection_server.MonitorConnectionServer
	backend                 EndpointBackend
	id                      *shortid.Shortid
	grpcServer              *grpc.Server
	registryClient          registry.NetworkServiceRegistryClient
	endpointName            string
}

func (nsme *nsmEndpoint) setupNSEServerConnection() (net.Listener, error) {
	c := nsme.configuration
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

func (nsme *nsmEndpoint) serve(listener net.Listener) {
	go func() {
		if err := nsme.grpcServer.Serve(listener); err != nil {
			logrus.Fatalf("nse: failed to start grpc server on socket %s with error: %v ", nsme.configuration.nsmClientSocket, err)
		}
	}()
}

func (nsme *nsmEndpoint) Start() error {
	nsme.grpcServer = grpc.NewServer()
	networkservice.RegisterNetworkServiceServer(nsme.grpcServer, nsme)

	listener, err := nsme.setupNSEServerConnection()

	if err != nil {
		logrus.Errorf("Unable to setup NSE")
		return err
	}

	// spawn the listnening thread
	nsme.serve(listener)

	// Registering NSE API, it will listen for Connection requests from NSM and return information
	// needed for NSE's dataplane programming.
	nse := &registry.NetworkServiceEndpoint{
		NetworkServiceName: nsme.configuration.AdvertiseNseName,
		Payload:            "IP",
		Labels:             tools.ParseKVStringToMap(nsme.configuration.AdvertiseNseLabels, ",", "="),
	}
	registration := &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    nsme.configuration.AdvertiseNseName,
			Payload: "IP",
		},
		NetworkserviceEndpoint: nse,
	}

	nsme.registryClient = registry.NewNetworkServiceRegistryClient(nsme.grpcClient)
	registeredNSE, err := nsme.registryClient.RegisterNSE(nsme.context, registration)
	if err != nil {
		logrus.Fatalln("unable to register endpoint", err)
	}
	nsme.endpointName = registeredNSE.GetNetworkserviceEndpoint().GetEndpointName()
	logrus.Infof("NSE registered: %v", registeredNSE)
	logrus.Infof("NSE: channel has been successfully advertised, waiting for connection from NSM...")

	return nil
}

func (nsme *nsmEndpoint) Delete() error {
	// prepare and defer removing of the advertised endpoint
	removeNSE := &registry.RemoveNSERequest{
		EndpointName: nsme.endpointName,
	}
	_, err := nsme.registryClient.RemoveNSE(context.Background(), removeNSE)
	if err != nil {
		logrus.Errorf("Failed removing NSE: %v, with %v", removeNSE, err)
	}
	nsme.grpcServer.Stop()

	return err
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
