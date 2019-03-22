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

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/sirupsen/logrus"
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
func (nsmcl *NsmClientList) Connect(name, mechanism, description string) error {
	for idx := range nsmcl.clients {
		entry := &nsmcl.clients[idx]
		conn, err := entry.client.Connect(name+strconv.Itoa(idx), mechanism, description)
		if err != nil {
			return err
		}
		entry.connections = append(entry.connections, conn)
	}
	return nil
}

// Close terminates all connections establised by Connect
func (nsmcl *NsmClientList) Close() error {
	for i := range nsmcl.clients {
		entry := &nsmcl.clients[i]
		for _, connection := range entry.connections {
			err := entry.client.Close(connection)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Destroy terminates all clients
func (nsmcl *NsmClientList) Destroy() error {
	for i := range nsmcl.clients {
		entry := &nsmcl.clients[i]
		err := entry.client.Destroy()
		if err != nil {
			return err
		}
	}
	return nil
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
				nsmClientListEntry{
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
		configuration = common.NSConfigurationFromUrl(configuration, url)
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
