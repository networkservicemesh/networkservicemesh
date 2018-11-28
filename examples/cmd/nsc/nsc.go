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
	"time"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"

	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
)

func main() {
	// For NSC to program container's dataplane, container's linux namespace must be sent to NSM
	netns, err := tools.GetCurrentNS()
	if err != nil {
		logrus.Fatalf("nsc: failed to get a linux namespace with error: %+v, exiting...", err)
		os.Exit(101)
	}
	logrus.Infof("Starting NSC, linux namespace: %s...", netns)

	// Init related activities start here
	logrus.Info("Connecting to nsm server on socket")
	nsmConnectionClient, conn, err := nsmd.NewNetworkServiceClient()
	if err != nil {
		logrus.Fatalf("nsc: failed to connect with NSMD: %+v, exiting...", err)
		os.Exit(101)
	}
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "icmp-responder",
			Context: map[string]string{
				"requires": "src_ip,dst_ip",
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*connection.Mechanism{
			{
				Type: connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					connection.NetNsInodeKey:    netns,
					connection.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	for ; true; <-time.After(5 * time.Second) {
		logrus.Infof("Sending request %v", request)
		reply, err := nsmConnectionClient.Request(context.Background(), request)

		if err != nil {
			logrus.Errorf("failure to request connection with error: %+v", err)
			continue
		}
		logrus.Infof("Received reply: %v", reply)
		break
		// Init related activities ends here
	}
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
