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

package main

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
)

const (
	defaultVPPAgentEndpoint = "localhost:9112"
)

type vppagentComposite struct {
	endpoint.BaseCompositeEndpoint
	vppAgentEndpoint string
	workspace        string
}

func (ns *vppagentComposite) GetOpaque(interface{}) interface{} {
	return nil
}

func (ns *vppagentComposite) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if ns.GetNext() == nil {
		logrus.Fatal("Should have Next set")
	}

	incoming, err := ns.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	err = ns.CreateVppInterface(ctx, incoming, ns.workspace)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return incoming, nil
}

func (ns *vppagentComposite) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

// NewVppAgentComposite creates a new VPP Agent composite
func newVppAgentComposite(configuration *common.NSConfiguration) *vppagentComposite {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	newVppAgentComposite := &vppagentComposite{
		vppAgentEndpoint: defaultVPPAgentEndpoint,
		workspace:        configuration.Workspace,
	}
	newVppAgentComposite.Reset()

	return newVppAgentComposite
}
