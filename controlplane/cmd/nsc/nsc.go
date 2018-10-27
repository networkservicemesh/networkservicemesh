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
	"os"

	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmdapi"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
)

func setupClientConnection() (string, error) {
	serverSocket := nsmd.ServerSock
	logrus.Infof("Connecting to nsmd on socket: %s...", serverSocket)
	if _, err := os.Stat(serverSocket); err != nil {
		return "", err
	}
	conn, err := tools.SocketOperationCheck(serverSocket)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	logrus.Info("Requesting nsmd for client connection...")
	client := nsmdapi.NewNSMDClient(conn)
	reply, err := client.RequestClientConnection(context.Background(), &nsmdapi.ClientConnectionRequest{})
	if err != nil {
		return "", err
	}
	logrus.Infof("nsmd provided socket %s for client operations...", reply.SocketLocation)
	return reply.SocketLocation, nil
}

func main() {
	// For NSC to program container's dataplane, container's linux namespace must be sent to NSM
	linuxNS, err := tools.GetCurrentNS()
	if err != nil {
		logrus.Fatalf("nsc: failed to get a linux namespace with error: %+v, exiting...", err)
		os.Exit(1)
	}
	logrus.Infof("Starting NSC, linux namespace: %s...", linuxNS)

	clientSocket, err := setupClientConnection()
	if err != nil {
		logrus.Fatalf("nsc: failed set up client connection, error: %+v, exiting...", err)
		os.Exit(1)
	}

	logrus.Infof("Connecting to nsm server on socket: %s...", clientSocket)
	if _, err := os.Stat(clientSocket); err != nil {
		logrus.Errorf("nsc: failure to access nsm socket at %s with error: %+v, exiting...", clientSocket, err)
		os.Exit(1)
	}

	conn, err := tools.SocketOperationCheck(clientSocket)
	if err != nil {
		logrus.Fatalf("nsm client: failure to communicate with the socket %s with error: %+v", clientSocket, err)
		os.Exit(1)
	}
	defer conn.Close()

	// Init related activities start here
	nsmConnectionClient := nsmconnect.NewClientConnectionClient(conn)

	_, err = nsmConnectionClient.RequestConnection(context.Background(), &nsmconnect.ConnectionRequest{
		RequestId:          linuxNS,
		LinuxNamespace:     linuxNS,
		NetworkServiceName: "gold-network",
		LocalMechanisms: []*common.LocalMechanism{
			&common.LocalMechanism{
				Type: common.LocalMechanismType_KERNEL_INTERFACE,
			},
		},
	})

	if err != nil {
		logrus.Fatalf("failure to request connection with error: %+v", err)
		os.Exit(1)
	}

	// Init related activities ends here
	logrus.Info("nsm client: initialization is completed successfully, exiting...")
}
