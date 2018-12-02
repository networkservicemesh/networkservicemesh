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
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	// NetworkServiceName defines Network Service Name the NSE is serving for
	NetworkServiceNameEnv   = "NETWORK_SERVICE"
	DefaultVPPAgentEndpoint = "localhost:9112"
)

func main() {
	// Capture signals to cleanup before exiting
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	nsmServerSocket, ok := os.LookupEnv(nsmd.NsmServerSocketEnv)
	if !ok {
		logrus.Fatalf("Error getting %v: %v", nsmd.NsmServerSocketEnv, ok)
	}
	logrus.Infof("nsmServerSocket: %s", nsmServerSocket)

	nsmClientSocket, ok := os.LookupEnv(nsmd.NsmClientSocketEnv)
	if !ok {
		logrus.Fatalf("Error getting %v: %v", nsmd.NsmClientSocketEnv, ok)
	}
	logrus.Infof("nsmClientSocket: %s", nsmClientSocket)

	workspace, ok := os.LookupEnv(nsmd.WorkspaceEnv)
	if !ok {
		logrus.Fatalf("Error getting %v: %v", nsmd.WorkspaceEnv, ok)
	}
	logrus.Infof("workspace: %s", workspace)

	networkServiceName, ok := os.LookupEnv(NetworkServiceNameEnv)
	if !ok {
		logrus.Fatalf("Error getting %v: %v", NetworkServiceNameEnv, ok)
	}
	logrus.Infof("Network Service Name: %s", networkServiceName)

	// For NSE to program container's dataplane, container's linux namespace must be sent to NSM
	linuxNS, err := tools.GetCurrentNS()
	if err != nil {
		logrus.Fatalf("nse: failed to get a linux namespace with error: %v, exiting...", err)
	}
	logrus.Infof("Starting NSE, linux namespace: %s", linuxNS)

	if err := tools.SocketCleanup(nsmClientSocket); err != nil {
		logrus.Fatalf("nse: failure to cleanup stale socket %s with error: %v", nsmClientSocket, err)
	}

	logrus.Infof("nse: listening socket %s", nsmClientSocket)
	connectionServer, err := net.Listen("unix", nsmClientSocket)
	if err != nil {
		logrus.Fatalf("nse: failure to listen on a socket %s with error: %v", nsmClientSocket, err)
	}
	grpcServer := grpc.NewServer()

	// Registering NSE API, it will listen for Connection requests from NSM and return information
	// needed for NSE's dataplane programming.

	go func() {
		if err := grpcServer.Serve(connectionServer); err != nil {
			logrus.Fatalf("nse: failed to start grpc server on socket %s with error: %v ", nsmClientSocket, err)
		}
	}()
	// Check if the socket of Endpoint Connection Server is operation
	testSocket, err := tools.SocketOperationCheck(nsmServerSocket)
	if err != nil {
		logrus.Fatalf("nse: failure to communicate with the nsm on socket %s with error: %v", nsmServerSocket, err)
	}
	testSocket.Close()

	// NSE connection server is ready and now endpoints can be advertised to NSM

	if _, err := os.Stat(nsmServerSocket); err != nil {
		logrus.Fatalf("nse: failure to access nsm socket at %s with error: %+v, exiting...", nsmServerSocket, err)
	}

	conn, err := tools.SocketOperationCheck(nsmServerSocket)
	if err != nil {
		logrus.Fatalf("nse: failure to communicate with the registrySocket %s with error: %+v", nsmServerSocket, err)
	}
	defer conn.Close()
	logrus.Infof("nsm: connection to nsm server on socket: %s succeeded.", nsmServerSocket)

	registryConnection := registry.NewNetworkServiceRegistryClient(conn)
	clientConnection := networkservice.NewNetworkServiceClient(conn)

	nseConn := New(networkServiceName, DefaultVPPAgentEndpoint, workspace, clientConnection)
	networkservice.RegisterNetworkServiceServer(grpcServer, nseConn)

	nse := &registry.NetworkServiceEndpoint{
		NetworkServiceName: networkServiceName,
		Payload:            "IP",
		Labels: map[string]string{
			"app": "firewall", // TODO - make these ENV configurable
		},
	}
	registration := &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    networkServiceName,
			Payload: "IP",
		},
		NetworkserviceEndpoint: nse,
	}

	registeredNSE, err := registryConnection.RegisterNSE(context.Background(), registration)
	if err != nil {
		logrus.Fatalln("unable to register endpoint", err)
	}
	logrus.Infof("NSE registered: %v", registeredNSE)

	// prepare and defer removing of the advertised endpoint
	removeNSE := &registry.RemoveNSERequest{
		EndpointName: registeredNSE.GetNetworkserviceEndpoint().GetEndpointName(),
	}

	defer registryConnection.RemoveNSE(context.Background(), removeNSE)
	defer grpcServer.Stop()

	logrus.Infof("nse: channel has been successfully advertised, waiting for connection from NSM...")

	select {
	case <-c:
		logrus.Infof("Closing %v", networkServiceName)
	}
}
