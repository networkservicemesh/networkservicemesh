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
	"fmt"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/examples/cmd/nsc/utils"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

const (
	attemptsMax = 10
)

func newNetworkServiceRequest(networkServiceName string, intf string, netns string) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: networkServiceName,
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
					connection.InterfaceNameKey: intf,
				},
			},
		},
	}
}

func main() {
	// For NSC to program container's dataplane, container's linux namespace must be sent to NSM
	netns, err := tools.GetCurrentNS()
	if err != nil {
		logrus.Fatalf("nsc: failed to get a linux namespace with error: %+v, exiting...", err)
		os.Exit(101)
	}
	logrus.Infof("Starting NSC, linux namespace: %s...", netns)

	networkServices, ok := os.LookupEnv("NETWORK_SERVICES")
	if !ok {
		logrus.Infof("nsc: no services to connect, exiting...")
		os.Exit(0)
	}

	// Init related activities start here
	logrus.Info("Connecting to nsm server on socket")
	nsmConnectionClient, conn, err := nsmd.NewNetworkServiceClient()
	if err != nil {
		logrus.Fatalf("nsc: failed to connect with NSMD: %+v, exiting...", err)
		os.Exit(101)
	}
	defer conn.Close()

	nsConfig := utils.ParseNetworkServices(networkServices)
	var requests []*networkservice.NetworkServiceRequest

	for ns, intf := range nsConfig {
		requests = append(requests, newNetworkServiceRequest(ns, intf, netns))
	}

	errorCh := make(chan error)
	waitCh := make(chan struct{})

	go func() {
		var wg sync.WaitGroup

		for _, r := range requests {
			wg.Add(1)
			go func(r *networkservice.NetworkServiceRequest, errorCh chan<- error) {
				logrus.Infof("start goroutine for requesting connection with %v", r.Connection.NetworkService)
				defer wg.Done()
				attempt := 0
				for attempt < attemptsMax {
					<-time.Tick(time.Second)

					attempt++
					logrus.Infof("Sending request %v", r)
					reply, err := nsmConnectionClient.Request(context.Background(), r)
					if err != nil {
						logrus.Errorf("failure to request connection with error: %+v", err)
						continue
					}
					logrus.Infof("Received reply: %v", reply)
					return
				}
				errorCh <- fmt.Errorf("unable to setup connection with %v after %v attempts",
					r.Connection.NetworkService, attempt)
			}(r, errorCh)
		}

		wg.Wait()
		close(waitCh)
	}()

	select {
	case err := <-errorCh:
		logrus.Error(err)
		os.Exit(1)
	case <-waitCh:
		logrus.Info("nsm client: initialization is completed successfully")
	}
}
