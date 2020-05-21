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
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"

	pkgerrors "github.com/pkg/errors"
)

type nsmClientListEntry struct {
	client      *NsmClient
	connections []*connection.Connection
}

// NsmClientList represents a set of clients
type NsmClientList struct {
	clients []nsmClientListEntry
}

// Connect will create new interfaces with the specified name and mechanism
func (l *NsmClientList) Connect(ctx context.Context, name, mechanism, description string) error {
	return l.ConnectRetry(ctx, name, mechanism, description, 0, 0)
}

// Connect will create new interfaces with the specified name and mechanism
func (l *NsmClientList) ConnectRetry(ctx context.Context, name, mechanism, description string, retryCount int, retryDelay time.Duration) error {
	for idx := range l.clients {
		entry := &l.clients[idx]
		if entry.client.NsmConnection.Configuration.PodName != "" &&
			entry.client.ClientLabels[connection.PodNameKey] == "" {
			entry.client.ClientLabels[connection.PodNameKey] = entry.client.NsmConnection.Configuration.PodName
		}
		if entry.client.NsmConnection.Configuration.Namespace != "" &&
			entry.client.ClientLabels[connection.NamespaceKey] == "" {
			entry.client.ClientLabels[connection.NamespaceKey] = entry.client.NsmConnection.Configuration.Namespace
		}
		conn, err := entry.client.ConnectRetry(ctx, name+strconv.Itoa(idx), mechanism, description, retryCount, retryDelay)
		if err != nil {
			return err
		}
		entry.connections = append(entry.connections, conn)
	}
	return nil
}

// Close terminates all connections establised by Connect
func (l *NsmClientList) Close(ctx context.Context) error {
	for i := range l.clients {
		entry := &l.clients[i]
		for _, connection := range entry.connections {
			err := entry.client.Close(ctx, connection)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Destroy terminates all clients
func (l *NsmClientList) Destroy(ctx context.Context) error {
	var err error
	for i := range l.clients {
		entry := &l.clients[i]
		derr := entry.client.Destroy(ctx)
		if derr != nil {
			err = pkgerrors.Wrap(err, derr.Error())
		}
	}
	return err
}

// NewNSMClientList creates a new list of clients
func NewNSMClientList(ctx context.Context, configuration *common.NSConfiguration) (*NsmClientList, error) {
	annotationValue := os.Getenv(AnnotationEnv)
	if annotationValue == "" {
		client, err := NewNSMClient(ctx, configuration)
		if err != nil {
			return nil, err
		}
		return &NsmClientList{
			clients: []nsmClientListEntry{
				{
					client: client}},
		}, nil
	}

	urls, err := tools.ParseAnnotationValue(annotationValue)
	if err != nil {
		logrus.Errorf("Bad annotation value: %v", err)
		return nil, err
	}

	var clients []nsmClientListEntry
	for _, url := range urls {
		configuration = configuration.FromNSUrl(url)
		client, err := NewNSMClient(ctx, configuration)
		if err != nil {
			return nil, err
		}
		clients = append(clients, nsmClientListEntry{
			client: client,
		})
	}
	return &NsmClientList{
		clients: clients,
	}, nil
}

// Clients returns NsmClients of NsmClientList
func (l *NsmClientList) Clients() []*NsmClient {
	var result []*NsmClient
	for _, c := range l.clients {
		result = append(result, c.client)
	}
	return result
}
