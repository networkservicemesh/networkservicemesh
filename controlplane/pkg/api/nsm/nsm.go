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

// Package nsm provides basic nsm interfaces
package nsm

import (
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/properties"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"
	crossconnect_monitor "github.com/networkservicemesh/networkservicemesh/sdk/monitor/crossconnect"

	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

// ClientConnection is an interface for client connection
type ClientConnection interface {
	GetID() string
	GetConnectionSource() *connection.Connection
	GetConnectionDestination() *connection.Connection
	GetNetworkService() string
}

// NetworkServiceClient is an interface for network service client
type NetworkServiceClient interface {
	Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error)
	Close(ctx context.Context, connection *connection.Connection) error

	Cleanup() error
}

// HealState - keep the cause of healing process
type HealState int32

const (
	// HealStateDstDown is a case when destination is down: we need to restore it and re-program local Forwarder.
	HealStateDstDown HealState = 1
	// HealStateSrcDown is a case when source is down: most probable will not happen yet.
	HealStateSrcDown HealState = 2
	// HealStateForwarderDown is a case when local Forwarder is down: we need to heal NSE/Remote NSM and local Forwarder.
	HealStateForwarderDown HealState = 3
	// HealStateDstUpdate is a case when destination is updated: we need to re-program local Forwarder.
	HealStateDstUpdate HealState = 4
	// HealStateDstNmgrDown is a case when destination and/or Remote NSM is down: we need to heal NSE/Remote NSM.
	HealStateDstNmgrDown HealState = 5
)

// NetworkServiceRequestManager - allow to provide local and remote service interfaces.
type NetworkServiceRequestManager interface {
	LocalManager(clientConnection ClientConnection) networkservice.NetworkServiceServer
	RemoteManager() networkservice.NetworkServiceServer
}

// NetworkServiceHealProcessor - perform Healing operations
type NetworkServiceHealProcessor interface {
	Heal(ctx context.Context, clientConnection ClientConnection, healState HealState)
	CloseConnection(ctx context.Context, clientConnection ClientConnection) error
}

// MonitorManager is an interface to provide access to different monitors
type MonitorManager interface {
	CrossConnectMonitor() crossconnect_monitor.MonitorServer
	LocalConnectionMonitor(workspace string) connectionmonitor.MonitorServer
}

//NetworkServiceManager - hold useful nsm structures
type NetworkServiceManager interface {
	GetHealProperties() *properties.Properties
	WaitForForwarder(ctx context.Context, duration time.Duration) error
	RemoteConnectionLost(ctx context.Context, clientConnection ClientConnection)
	NotifyRenamedEndpoint(nseOldName, nseNewName string)
	// Getters
	NseManager() NetworkServiceEndpointManager
	SetRemoteServer(server networkservice.NetworkServiceServer)

	Model() model.Model

	NetworkServiceHealProcessor
	ServiceRegistry() serviceregistry.ServiceRegistry
	RestoreConnections(xcons []*crossconnect.CrossConnect, forwarder string, manager MonitorManager)
}

//NetworkServiceEndpointManager - manages endpoints, TODO: Will be removed in next PRs.
type NetworkServiceEndpointManager interface {
	GetEndpoint(ctx context.Context, requestConnection *connection.Connection, ignoreEndpoints map[registry.EndpointNSMName]*registry.NSERegistration) (*registry.NSERegistration, error)
	CreateNSEClient(ctx context.Context, endpoint *registry.NSERegistration) (NetworkServiceClient, error)
	IsLocalEndpoint(endpoint *registry.NSERegistration) bool
	CheckUpdateNSE(ctx context.Context, reg *registry.NSERegistration) bool
}
