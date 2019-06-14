// Copyright 2018, 2019 VMware, Inc.
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

package common

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/security/certificate"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

// NsmConnection is a NSM manager connection
type NsmConnection struct {
	sync.RWMutex
	Context       context.Context
	Configuration *NSConfiguration
	GrpcClient    *grpc.ClientConn
	NsClient      networkservice.NetworkServiceClient
}

// Close terminates the connection
func (nsmc *NsmConnection) Close() error {
	return nsmc.GrpcClient.Close()
}

// NewNSMConnection creates a NsmConnection
func NewNSMConnection(ctx context.Context, configuration *NSConfiguration) (*NsmConnection, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	conn := NsmConnection{
		Context:       ctx,
		Configuration: configuration,
	}
	// For NSE to program container's dataplane, container's linux namespace must be sent to NSM
	// linuxNS, err := tools.GetCurrentNS()
	// if err != nil {
	// 	logrus.Fatalf("nse: failed to get a linux namespace with error: %v, exiting...", err)
	// }
	// logrus.Infof("Starting NSE, linux namespace: %s", linuxNS)

	// NSE connection server is ready and now endpoints can be advertised to NSM
	certObtainer := security.NewSpireCertObtainer("/run/spire/sockets/agent.sock", 5*time.Second)
	cm := security.NewCertificateManager(certObtainer)

	cred, err := cm.ClientCredentials()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	// Check if the socket of Endpoint Connection Server is operable
	testSocket, err := tools.SocketOperationCheckSecure("unix:"+tools.SocketPath(configuration.NsmServerSocket), cred)
	if err != nil {
		logrus.Errorf("nse: failure to communicate with the nsm on socket %s with error: %v", configuration.NsmServerSocket, err)
		return nil, err
	}
	testSocket.Close()

	if _, err := os.Stat(configuration.NsmServerSocket); err != nil {
		logrus.Errorf("nse: failure to access nsm socket at %s with error: %+v, exiting...", configuration.NsmServerSocket, err)
		return nil, err
	}

	conn.GrpcClient, err = tools.SocketOperationCheckSecure("unix:"+tools.SocketPath(configuration.NsmServerSocket), cred)
	if err != nil {
		logrus.Errorf("nse: failure to communicate with the registrySocket %s with error: %+v", configuration.NsmServerSocket, err)
		return nil, err
	}
	logrus.Infof("nsm: connection to nsm server on socket: %s succeeded.", configuration.NsmServerSocket)

	conn.NsClient = networkservice.NewNetworkServiceClient(conn.GrpcClient)

	return &conn, nil
}
