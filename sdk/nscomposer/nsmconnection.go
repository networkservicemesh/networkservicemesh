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
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type nsmConnection struct {
	sync.RWMutex
	context       context.Context
	configuration *NSConfiguration
	grpcClient    *grpc.ClientConn
	nsClient      networkservice.NetworkServiceClient
}

func (nsmc *nsmConnection) Close() error {
	return nsmc.grpcClient.Close()
}

func newNSMConnection(ctx context.Context, configuration *NSConfiguration) (*nsmConnection, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	conn := nsmConnection{
		context:       ctx,
		configuration: configuration,
	}
	// For NSE to program container's dataplane, container's linux namespace must be sent to NSM
	// linuxNS, err := tools.GetCurrentNS()
	// if err != nil {
	// 	logrus.Fatalf("nse: failed to get a linux namespace with error: %v, exiting...", err)
	// }
	// logrus.Infof("Starting NSE, linux namespace: %s", linuxNS)

	// NSE connection server is ready and now endpoints can be advertised to NSM

	// Check if the socket of Endpoint Connection Server is operable
	testSocket, err := tools.SocketOperationCheck(configuration.nsmServerSocket)
	if err != nil {
		logrus.Errorf("nse: failure to communicate with the nsm on socket %s with error: %v", configuration.nsmServerSocket, err)
		return nil, err
	}
	testSocket.Close()

	if _, err := os.Stat(configuration.nsmServerSocket); err != nil {
		logrus.Errorf("nse: failure to access nsm socket at %s with error: %+v, exiting...", configuration.nsmServerSocket, err)
		return nil, err
	}

	conn.grpcClient, err = tools.SocketOperationCheck(configuration.nsmServerSocket)
	if err != nil {
		logrus.Errorf("nse: failure to communicate with the registrySocket %s with error: %+v", configuration.nsmServerSocket, err)
		return nil, err
	}
	logrus.Infof("nsm: connection to nsm server on socket: %s succeeded.", configuration.nsmServerSocket)

	conn.nsClient = networkservice.NewNetworkServiceClient(conn.grpcClient)

	return &conn, nil
}
