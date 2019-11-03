// Copyright (c) 2019 Cisco Systems, Inc.
//
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
package nsmd

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

type networkServiceDiscoveryServer struct {
	serviceRegistry serviceregistry.ServiceRegistry
}

func NewNetworkServiceDiscoveryServer(serviceRegistry serviceregistry.ServiceRegistry) registry.NetworkServiceDiscoveryServer {
	return &networkServiceDiscoveryServer{serviceRegistry: serviceRegistry}
}

func (n networkServiceDiscoveryServer) FindNetworkService(ctx context.Context, find *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	client, err := n.serviceRegistry.DiscoveryClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.FindNetworkService(ctx, find)
}
