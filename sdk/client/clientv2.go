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
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"os"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

const (
	annotationEnv = "NS_NETWORKSERVICEMESH_IO"
)

type NsmClientV2 struct {
	clients     []*NsmClient
	connections []*connection.Connection
}

func configFromUrl(configuration *common.NSConfiguration, url *tools.NsUrl) *common.NSConfiguration {
	var conf common.NSConfiguration
	if configuration != nil {
		conf = *configuration
	}
	conf.OutgoingNscName = url.NsName
	var labels strings.Builder
	separator := false
	for k, v := range url.Params {
		if separator {
			labels.WriteRune(',')
		} else {
			separator = true
		}
		labels.WriteString(k)
		labels.WriteRune('=')
		labels.WriteString(v[0])
	}
	conf.OutgoingNscLabels = labels.String()
	return &conf
}

func NewNSMClientV2(ctx context.Context, configuration *common.NSConfiguration) (*NsmClientV2, error) {
	annotationValue := os.Getenv(annotationEnv)
	if len(annotationValue) == 0 {
		client, err := NewNSMClient(ctx, configuration)
		if err != nil {
			return nil, err
		}
		return &NsmClientV2{
			clients: []*NsmClient{client},
		}, nil
	}

	urls, err := tools.ParseAnnotationValue(annotationValue)
	if err != nil {
		logrus.Errorf("Bad annotation value: %v", err)
		return nil, err
	}

	var clients []*NsmClient
	for _, url := range urls {
		client, err := NewNSMClient(ctx, configFromUrl(configuration, url))
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return &NsmClientV2{
		clients: clients,
	}, nil
}

func (nsmc *NsmClientV2) Connect(name, mechanism, description string) error {
	for _, client := range nsmc.clients {
		conn, err := client.Connect(name, mechanism, description)
		if err != nil {
			return err
		}
		nsmc.connections = append(nsmc.connections, conn)
	}
	return nil
}
