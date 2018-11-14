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
	"os/signal"
	"sync"
	"syscall"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/networkservice"

	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
)

func main() {
	// For NSC to program container's dataplane, container's linux namespace must be sent to NSM
	netns, err := tools.GetCurrentNS()
	if err != nil {
		logrus.Fatalf("nsc: failed to get a linux namespace with error: %+v, exiting...", err)
	}
	logrus.Infof("Starting NSC, linux namespace: %s...", netns)

	nsmServerSocket, _ := os.LookupEnv(nsmd.NsmServerSocketEnv)
	// TODO handle case where env variable is not set

	logrus.Infof("Connecting to nsm server on socket: %s...", nsmServerSocket)
	if _, err := os.Stat(nsmServerSocket); err != nil {
		logrus.Fatalf("nsc: failure to access nsm socket at %s with error: %+v, exiting...", nsmServerSocket, err)
	}

	conn, err := tools.SocketOperationCheck(nsmServerSocket)
	if err != nil {
		logrus.Fatalf("nsm client: failure to communicate with the socket %s with error: %+v", nsmServerSocket, err)
	}
	defer conn.Close()

	// Init related activities start here
	nsmConnectionClient := networkservice.NewNetworkServiceClient(conn)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "icmp-responder",
			ConnectionContext: &networkservice.ConnectionContext{
				ConnectionContext: map[string]string{
					"requires": "src_ip,dst_ip",
				},
			},
			Labels: make(map[string]string),
		},
		LocalMechanismPreference: []*common.LocalMechanism{
			{
				Type:       common.LocalMechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{nsmd.LocalMechanismParameterNetNsInodeKey: netns},
			},
		},
	}

	logrus.Infof("Sending request %v", request)
	reply, err := nsmConnectionClient.Request(context.Background(), request)

	if err != nil {
		logrus.Fatalln("failure to request connection with error: %+v", err)
	}
	logrus.Infof("Received reply: %v", reply)

	// Init related activities ends here
	logrus.Info("nsm client: initialization is completed successfully, wait for Ctrl+C...")

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		wg.Done()
	}()
	wg.Wait()
}
