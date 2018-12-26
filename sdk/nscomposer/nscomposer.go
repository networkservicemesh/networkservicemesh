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
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func runNSEndpoint(ctx context.Context, configuration *NSConfiguration, backend EndpointBackend) (*grpc.Server, registry.NetworkServiceRegistryClient, *registry.RemoveNSERequest, error) {

	nsmEndpoint, err := NewNSMEndpoint(
		ctx,
		configuration,
		backend)

	// Registering NSE API, it will listen for Connection requests from NSM and return information
	// needed for NSE's dataplane programming.
	grpcServer := grpc.NewServer()
	networkservice.RegisterNetworkServiceServer(grpcServer, nsmEndpoint)

	connectionServer, _ := nsmEndpoint.setupNSEServerConnection()

	go func() {
		if err := grpcServer.Serve(connectionServer); err != nil {
			logrus.Fatalf("nse: failed to start grpc server on socket %s with error: %v ", nsmEndpoint.configuration.nsmClientSocket, err)
		}
	}()

	nse := &registry.NetworkServiceEndpoint{
		NetworkServiceName: nsmEndpoint.configuration.AdvertiseNseName,
		Payload:            "IP",
		Labels:             tools.ParseKVStringToMap(nsmEndpoint.configuration.AdvertiseNseLabels, ",", "="),
	}
	registration := &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    nsmEndpoint.configuration.AdvertiseNseName,
			Payload: "IP",
		},
		NetworkserviceEndpoint: nse,
	}

	registryConnection := registry.NewNetworkServiceRegistryClient(nsmEndpoint.grpcClient)
	registeredNSE, err := registryConnection.RegisterNSE(context.Background(), registration)
	if err != nil {
		logrus.Fatalln("unable to register endpoint", err)
	}
	logrus.Infof("NSE registered: %v", registeredNSE)

	// prepare and defer removing of the advertised endpoint
	removeNSE := &registry.RemoveNSERequest{
		EndpointName: registeredNSE.GetNetworkserviceEndpoint().GetEndpointName(),
	}

	logrus.Infof("nse: channel has been successfully advertised, waiting for connection from NSM...")

	return grpcServer, registryConnection, removeNSE, nil
}

func NsComposerMain(ctx context.Context, configuration *NSConfiguration, backend EndpointBackend) {

	grpcServer, registryConnection, removeNSE, err := runNSEndpoint(ctx, configuration, backend)

	if err != nil {
		logrus.Fatalf("%v", err)
	}

	defer registryConnection.RemoveNSE(context.Background(), removeNSE)
	defer grpcServer.Stop()

	// Capture signals to cleanup before exiting
	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		<-c
		wg.Done()
		logrus.Infof("Deregistering %s at %v", configuration.AdvertiseNseName, registryConnection)
		logrus.Infof("Stopping %v", grpcServer)
	}()
	wg.Wait()

}
