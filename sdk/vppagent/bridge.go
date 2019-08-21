// Copyright 2019 VMware, Inc.
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

package vppagent

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/models/vpp"
	l2 "github.com/ligato/vpp-agent/api/models/vpp/l2"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

//
type bridgeConnect struct {
	workspace  string
	bridgeName string
}

// Request - plugs interface into a Bridge domain
func (vbc *bridgeConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	err := vbc.insertInterfaceIntoBridge(ctx, request.GetConnection())
	if err != nil {
		endpoint.Log(ctx).Error(err)
		return nil, err
	}
	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Request(ctx, request)
	}
	return request.GetConnection(), nil
}

// Close - disconnect interface from Bridge Domain
func (vbc *bridgeConnect) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	err := vbc.insertInterfaceIntoBridge(ctx, conn)
	if err != nil {
		endpoint.Log(ctx).Error(err)
		return &empty.Empty{}, err
	}
	if endpoint.Next(ctx) != nil {
		if _, err := endpoint.Next(ctx).Close(ctx, conn); err != nil {
			return &empty.Empty{}, nil
		}
	}
	return &empty.Empty{}, nil
}

// NewBridgeConnect creates a new Bridge Endpoint
func NewBridgeConnect(configuration *common.NSConfiguration, bridgeName string) networkservice.NetworkServiceServer {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	bridge := &bridgeConnect{
		workspace:  configuration.Workspace,
		bridgeName: bridgeName,
	}
	return bridge
}

func (vbc *bridgeConnect) insertInterfaceIntoBridge(ctx context.Context, conn *connection.Connection) error {
	cfg := Config(ctx)
	conMap := ConnectionMap(ctx)
	if conMap[conn] == nil {
		return fmt.Errorf("BridgeConnect - context missing ConnectionMap entry for %v", conn)
	}

	if cfg.VppConfig == nil {
		cfg.VppConfig = &vpp.ConfigData{}
	}
	cfg.VppConfig.BridgeDomains = append(cfg.VppConfig.BridgeDomains, &l2.BridgeDomain{
		Name:                vbc.bridgeName,
		Flood:               false,
		UnknownUnicastFlood: false,
		Forward:             true,
		Learn:               true,
		ArpTermination:      false,
		Interfaces: []*l2.BridgeDomain_Interface{
			{
				Name:                    conMap[conn].GetName(),
				BridgedVirtualInterface: false,
			},
		},
	},
	)
	return nil
}
