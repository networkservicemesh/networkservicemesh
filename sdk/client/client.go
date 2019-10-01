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
	"fmt"
	"io"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/opentracing/opentracing-go/log"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

const (
	// ConnectTimeout - a default connection timeout
	ConnectTimeout = 15 * time.Second
	// ConnectionRetry - A number of retries for establish a network service, default == 10
	ConnectionRetry = 10
	// RequestDelay - A delay between attempts, default = 5sec
	RequestDelay = time.Second * 5
)

// NsmClient is the NSM client struct
type NsmClient struct {
	*common.NsmConnection
	OutgoingNscName     string
	OutgoingNscLabels   map[string]string
	OutgoingConnections []*connection.Connection
	tracerCloser        io.Closer
}

// Connect with no retry and delay
func (nsmc *NsmClient) Connect(ctx context.Context, name, mechanism, description string) (*connection.Connection, error) {
	return nsmc.ConnectRetry(ctx, name, mechanism, description, 1, 0)
}

// Connect implements the business logic
func (nsmc *NsmClient) ConnectRetry(ctx context.Context, name, mechanism, description string, retryCount int, retryDelay time.Duration) (*connection.Connection, error) {

	var span opentracing.Span
	if opentracing.IsGlobalTracerRegistered() {
		span, ctx = opentracing.StartSpanFromContext(ctx, "nsmClient.Connect")
		defer span.Finish()
	}

	logger := common.LogFromSpan(span)

	logger.Infof("Initiating an outgoing connection.")
	nsmc.Lock()
	defer nsmc.Unlock()
	mechanismType := common.MechanismFromString(mechanism)
	outgoingMechanism, err := connection.NewMechanism(mechanismType, name, description)

	logger.Infof("Selected mechanism: %v", outgoingMechanism)

	if err != nil {
		logger.Errorf("Failure to prepare the outgoing mechanism preference with error: %+v", err)
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
	var outgoingConnection *connection.Connection
	maxRetry := retryCount
	for retryCount >= 0 {
		attempCtx, cancelProc := context.WithTimeout(ctx, ConnectTimeout)
		defer cancelProc()

		var attemptSpan opentracing.Span
		if opentracing.IsGlobalTracerRegistered() {
			attemptSpan, attempCtx = opentracing.StartSpanFromContext(attempCtx, fmt.Sprintf("nsmClient.Connect.attempt:%v", maxRetry-retryCount))
			defer attemptSpan.Finish()
		}

		attemptLogger := common.LogFromSpan(attemptSpan)
		attemptLogger.Infof("Requesting %v", outgoingRequest)
		outgoingConnection, err = nsmc.NsClient.Request(attempCtx, outgoingRequest)

		if err != nil {
			if attemptSpan != nil {
				attemptSpan.LogFields(log.Error(err))
				attemptSpan.Finish()
			}
			cancelProc()
			if retryCount == 0 {
				return nil, fmt.Errorf("nsm client: Failed to connect %v", err)
			} else {
				attemptLogger.Errorf("nsm client: Failed to connect %v. Retry attempts: %v Delaying: %v", err, retryCount, retryDelay)
			}
			retryCount--
			<-time.After(retryDelay)
			continue
		}
		break
	}
	logger.Infof("Success connection: %v", outgoingConnection)
	nsmc.OutgoingConnections = append(nsmc.OutgoingConnections, outgoingConnection)
	return outgoingConnection, nil
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

// Destroy - Destroy stops the whole module
func (nsmc *NsmClient) Destroy(_ context.Context) error {
	nsmc.Lock()
	defer nsmc.Unlock()

	if nsmc.tracerCloser != nil {
		err := nsmc.tracerCloser.Close()
		if err != nil {
			logrus.Errorf("failed to close opentracing context %v", err)
		}
	}

	return nsmc.NsmConnection.Close()
}

// NewNSMClient creates the NsmClient
func NewNSMClient(ctx context.Context, configuration *common.NSConfiguration) (*NsmClient, error) {
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}

	client := &NsmClient{
		OutgoingNscName:   configuration.OutgoingNscName,
		OutgoingNscLabels: tools.ParseKVStringToMap(configuration.OutgoingNscLabels, ",", "="),
	}

	client.tracerCloser = jaeger.InitJaeger("nsm-client")

	nsmConnection, err := common.NewNSMConnection(ctx, configuration)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		return nil, err
	}

	client.NsmConnection = nsmConnection

	return client, nil
}
