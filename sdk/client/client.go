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
	"io"
	"os"
	"strconv"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

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
	connectTimeout = 15 * time.Second
)

// NsmClient is the NSM client struct
type NsmClient struct {
	*common.NsmConnection
	OutgoingNscName     string
	OutgoingNscLabels   map[string]string
	OutgoingConnections []*connection.Connection
	tracerCloser        io.Closer
}

// Connect implements the business logic
func (nsmc *NsmClient) Connect(ctx context.Context, name, mechanism, description string) (*connection.Connection, error) {

	var span opentracing.Span
	if opentracing.GlobalTracer() != nil {
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
	start := time.Now()
	connectRetries := nsmc.getConnectRetries()
	for iteration := connectRetries; true; <-time.After(connectSleep) {
		outgoingConnection, err = nsmc.performRequest(ctx, start, iteration, outgoingRequest)

		if outgoingConnection != nil || err != nil {
			return outgoingConnection, err
		}
	}

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
		_ = nsmc.tracerCloser.Close()
	}

	return nsmc.NsmConnection.Close()
}

func (nsmc *NsmClient) getConnectRetries() int {
	connectRetries := connectRetries
	connectRetriesStr := os.Getenv(NSMConnectRetries)
	if connectRetriesStr != "" {
		value, err := strconv.ParseInt(connectRetriesStr, 10, 32)
		if err != nil || value < 1 {
			logrus.Fatalf("%s=%v env variable is invalid, should be number >=1", NSMConnectRetries, value)
		}
		connectRetries = int(value)
	}
	return connectRetries
}

// PerformRequest - perform a request
func (nsmc *NsmClient) performRequest(ctx context.Context, start time.Time, iteration int, outgoingRequest *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	var span opentracing.Span
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if opentracing.GlobalTracer() != nil {
		span, ctx = opentracing.StartSpanFromContext(ctx, "NsmClientPerformRequest")
		defer span.Finish()
		span.LogFields(log.Int("attempt", connectRetries-iteration))
	}
	logger := common.LogFromSpan(span)
	logger.Infof("Sending outgoing request %v", outgoingRequest)

	outgoingConnection, err := nsmc.NsClient.Request(ctx, outgoingRequest)
	if err != nil {
		logger.Errorf("failure to request connection with error: %+v", err)
		iteration--
		if iteration > 0 {
			return nil, nil
		}
		logger.Errorf("Connect failed after %v iterations and %v", connectRetries, time.Since(start))
		return nil, err
	}

	nsmc.OutgoingConnections = append(nsmc.OutgoingConnections, outgoingConnection)
	logger.Infof("Received outgoing connection after %v: %v", time.Since(start), outgoingConnection)
	return outgoingConnection, nil
}

// NewNSMClient creates the NsmClient
func NewNSMClient(ctx context.Context, configuration *common.NSConfiguration) (*NsmClient, error) {
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	client := &NsmClient{
		OutgoingNscName:   configuration.OutgoingNscName,
		OutgoingNscLabels: tools.ParseKVStringToMap(configuration.OutgoingNscLabels, ",", "="),
	}

	if configuration.TracerEnabled {
		if opentracing.GlobalTracer() == nil {
			tracer, closer := tools.InitJaeger("nsm-client")
			opentracing.SetGlobalTracer(tracer)
			client.tracerCloser = closer
		} else {
			logrus.Infof("Use already initialized global gracer")
		}
	}

	nsmConnection, err := common.NewNSMConnection(ctx, configuration)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		return nil, err
	}

	client.NsmConnection = nsmConnection

	return client, nil
}
