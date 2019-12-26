// Copyright (c) 2019 Cisco and/or its affiliates.
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

// Package serviceregistry -
package serviceregistry

import (
	"net"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/sid"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsmdapi"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/vni"
	forwarderapi "github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

type ApiRegistry interface {
	NewNSMServerListener() (net.Listener, error)
	NewPublicListener(nsmdAPIAddress string) (net.Listener, error)
}

/**
A method to obtain different connectivity mechanism for parts of model
*/
type ServiceRegistry interface {
	GetPublicAPI() string

	DiscoveryClient(ctx context.Context) (registry.NetworkServiceDiscoveryClient, error)
	NseRegistryClient(ctx context.Context) (registry.NetworkServiceRegistryClient, error)
	NsmRegistryClient(ctx context.Context) (registry.NsmRegistryClient, error)

	Stop()
	NSMDApiClient(ctx context.Context) (nsmdapi.NSMDClient, *grpc.ClientConn, error)
	ForwarderConnection(ctx context.Context, forwarder *model.Forwarder) (forwarderapi.ForwarderClient, *grpc.ClientConn, error)

	EndpointConnection(ctx context.Context, endpoint *model.Endpoint) (networkservice.NetworkServiceClient, *grpc.ClientConn, error)
	RemoteNetworkServiceClient(ctx context.Context, nsm *registry.NetworkServiceManager) (networkservice.NetworkServiceClient, *grpc.ClientConn, error)

	WaitForForwarderAvailable(ctx context.Context, model model.Model, timeout time.Duration) error

	VniAllocator() vni.VniAllocator
	SIDAllocator() sid.Allocator

	NewWorkspaceProvider() WorkspaceLocationProvider
}

type WorkspaceLocationProvider interface {
	HostBaseDir() string
	NsmBaseDir() string
	ClientBaseDir() string
	NsmServerSocket() string
	NsmClientSocket() string

	// A persistent file based NSE <-> Workspace registry.
	NsmNSERegistryFile() string
}
