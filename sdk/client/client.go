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

package client

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

const (
	connectRetries = 10
	connectSleep   = 5 * time.Second
	connectTimeout = 10 * time.Second
)

// NsmClient is the NSM client struct
type NsmClient struct {
	*common.NsmConnection
	OutgoingNscName     string
	OutgoingNscLabels   map[string]string
	OutgoingConnections []*connection.Connection
}

// Connect implements the business logic
func (nsmc *NsmClient) Connect(ctx context.Context, name, mechanism, description string) (*connection.Connection, error) {
	logrus.Infof("Initiating an outgoing connection.")
	nsmc.Lock()
	defer nsmc.Unlock()
	mechanismType := common.MechanismFromString(mechanism)
	outgoingMechanism, err := connection.NewMechanism(mechanismType, name, description)
	if err != nil {
		logrus.Errorf("Failure to prepare the outgoing mechanism preference with error: %+v", err)
		return nil, err
	}

	routes := []*connectioncontext.Route{}
	for _, r := range nsmc.Configuration.Routes {
		routes = append(routes, &connectioncontext.Route{
			Prefix: r,
		})
	}

	outgoingRequest := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: nsmc.Configuration.OutgoingNscName,
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					SrcIpRequired: true,
					DstIpRequired: true,
					SrcRoutes:     routes,
				},
			},
			Labels: nsmc.OutgoingNscLabels,
		},
		MechanismPreferences: []*connection.Mechanism{
			outgoingMechanism,
		},
	}

	return nsmc.PerformRequest(outgoingRequest)
	// var outgoingConnection *connection.Connection
	// for iteration := connectRetries; true; <-time.After(connectSleep) {
	// 	var err error
	// 	logrus.Infof("Sending outgoing request %v", outgoingRequest)

	// 	newCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	// 	defer cancel()
	// 	outgoingConnection, err = nsmc.NsClient.Request(newCtx, outgoingRequest)

	// 	if err != nil {
	// 		logrus.Errorf("failure to request connection with error: %+v", err)
	// 		iteration--
	// 		if iteration > 0 {
	// 			continue
	// 		}
	// 		logrus.Errorf("Connect failed after %v iterations and %v", connectRetries, time.Since(start))
	// 		return nil, err
	// 	}

	// 	nsmc.OutgoingConnections = append(nsmc.OutgoingConnections, outgoingConnection)
	// 	logrus.Infof("Received outgoing connection after %v: %v", time.Since(start), outgoingConnection)
	// 	break
	// }

	// return outgoingConnection, nil
}

// Close will terminate a particular connection
func (nsmc *NsmClient) Close(ctx context.Context, outgoingConnection *connection.Connection) error {
	nsmc.Lock()
	defer nsmc.Unlock()

	_, err := nsmc.NsClient.Close(ctx, outgoingConnection)

	arr := nsmc.OutgoingConnections
	for i, c := range arr {
		if c == outgoingConnection {
			copy(arr[i:], arr[i+1:])
			arr[len(arr)-1] = nil
			arr = arr[:len(arr)-1]
		}
	}
	return err
}

// Destroy stops the whole module
func (nsmc *NsmClient) Destroy() error {
	nsmc.Lock()
	defer nsmc.Unlock()

	for _, c := range nsmc.OutgoingConnections {
		nsmc.NsClient.Close(nsmc.Context, c)
	}
	nsmc.NsmConnection.Close()
	return nil
}

//PerformRequest - perform request
func (nsmc *NsmClient) PerformRequest(outgoingRequest *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	var outgoingConnection *connection.Connection
	start := time.Now()
	for iteration := connectRetries; true; <-time.After(connectSleep) {
		var err error
		logrus.Infof("Sending outgoing request %v", outgoingRequest)

		ctx, cancel := context.WithTimeout(nsmc.Context, connectTimeout)
		defer cancel()
		outgoingConnection, err = nsmc.NsClient.Request(ctx, outgoingRequest)

		if err != nil {
			logrus.Errorf("failure to request connection with error: %+v", err)
			iteration--
			if iteration > 0 {
				continue
			}
			logrus.Errorf("Connect failed after %v iterations and %v", connectRetries, time.Since(start))
			return nil, err
		}

		nsmc.OutgoingConnections = append(nsmc.OutgoingConnections, outgoingConnection)
		logrus.Infof("Received outgoing connection after %v: %v", time.Since(start), outgoingConnection)
		break
	}

	return outgoingConnection, nil
}

// NewNSMClient creates the NsmClient
func NewNSMClient(ctx context.Context, configuration *common.NSConfiguration) (*NsmClient, error) {
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	nsmConnection, err := common.NewNSMConnection(ctx, configuration)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		return nil, err
	}

	client := &NsmClient{
		NsmConnection:     nsmConnection,
		OutgoingNscName:   configuration.OutgoingNscName,
		OutgoingNscLabels: tools.ParseKVStringToMap(configuration.OutgoingNscLabels, ",", "="),
	}

	return client, nil
}
