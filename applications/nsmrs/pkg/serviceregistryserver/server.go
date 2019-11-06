// Copyright (c) 2019 Cisco and/or its affiliates.
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

// Package serviceregistryserver - NSM Registry Server
package serviceregistryserver

import (
	"context"
	"net"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

// ServiceRegistry - service starting NSE registry server
type ServiceRegistry interface {
	NewPublicListener(registryAPIAddress string) (net.Listener, error)
}

type serviceRegistry struct {
}

// NewPublicListener - Starts public listener for NSMRS services
func (*serviceRegistry) NewPublicListener(registryAPIAddress string) (net.Listener, error) {
	return net.Listen("tcp", registryAPIAddress)
}

// NewNSMDServiceRegistryServer - creates new service registry service
func NewNSMDServiceRegistryServer() ServiceRegistry {
	return &serviceRegistry{}
}

// New - creates new grcp server and registers NSE discovery and registry services
func New() *grpc.Server {
	server := tools.NewServer(context.Background())

	cache := NewNSERegistryCache()
	discovery := newDiscoveryService(cache)
	registryService := NewNseRegistryService(cache)
	registry.RegisterNetworkServiceDiscoveryServer(server, discovery)
	registry.RegisterNetworkServiceRegistryServer(server, registryService)

	cache.StartNSMDTracking()

	return server
}
