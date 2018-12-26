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
	"time"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
)

const (
	connectRetries = 10
	connectSleep   = 5 * time.Second
)

type ClientBackend interface {
	New() error
	Connect(ctx context.Context, connection *connection.Connection) error
	Close(ctx context.Context, connection *connection.Connection) error
}

type nsmClient struct {
	*nsmConnection
	outgoingNscName    string
	outgoingNscLabels  map[string]string
	name               string
	description        string
	mechanismType      connection.MechanismType
	outgoingConnection *connection.Connection
	backend            ClientBackend
}

func (nsmc *nsmClient) Connect() error {
	logrus.Infof("Initiating an outgoing connection.")

	outgoingMechanism, err := connection.NewMechanism(nsmc.mechanismType, nsmc.name, nsmc.description)
	if err != nil {
		logrus.Errorf("Failure to prepare the outgoing mechanism preference with error: %+v", err)
		return err
	}

	outgoingRequest := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: nsmc.configuration.OutgoingNscName,
			Context: &connectioncontext.ConnectionContext{
				SrcIpRequired: true,
				DstIpRequired: true,
			},
			Labels: nsmc.outgoingNscLabels,
		},
		MechanismPreferences: []*connection.Mechanism{
			outgoingMechanism,
		},
	}

	for iteration := connectRetries; true; <-time.After(connectSleep) {
		var err error
		logrus.Infof("Sending outgoing request %v", outgoingRequest)
		nsmc.outgoingConnection, err = nsmc.nsClient.Request(nsmc.context, outgoingRequest)

		if err != nil {
			logrus.Errorf("failure to request connection with error: %+v", err)
			iteration--
			if iteration > 0 {
				continue
			}
			logrus.Errorf("Connect failed after %v iterations", connectRetries)
		}

		logrus.Infof("Received outgoing connection: %v", nsmc.outgoingConnection)
		break
	}

	nsmc.backend.Connect(nsmc.context, nsmc.outgoingConnection)

	return nil
}

func (nsmc *nsmClient) GetConnection() *connection.Connection {
	return nsmc.outgoingConnection
}

func (nsmc *nsmClient) Close() error {
	nsmc.backend.Close(nsmc.context, nsmc.outgoingConnection)
	nsmc.nsClient.Close(nsmc.context, nsmc.outgoingConnection)
	nsmc.nsmConnection.Close()
	return nil
}

type dummyClientBackend struct{}

func (*dummyClientBackend) New() error { return nil }
func (*dummyClientBackend) Connect(ctx context.Context, connection *connection.Connection) error {
	return nil
}
func (*dummyClientBackend) Close(ctx context.Context, connection *connection.Connection) error {
	return nil
}

func NewNSMClient(ctx context.Context, configuration *NSConfiguration, backend ClientBackend) (*nsmClient, error) {
	if configuration == nil {
		configuration = &NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	if backend == nil {
		backend = &dummyClientBackend{}
	}

	nsmConnection, err := newNSMConnection(ctx, configuration)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		return nil, err
	}

	client := &nsmClient{
		nsmConnection:     nsmConnection,
		outgoingNscName:   configuration.OutgoingNscName,
		outgoingNscLabels: tools.ParseKVStringToMap(configuration.OutgoingNscLabels, ",", "="),
		mechanismType:     mechanismFromString(configuration.OutgoingNscMechanism),
		backend:           backend,
	}

	client.backend.New()
	return client, nil
}
