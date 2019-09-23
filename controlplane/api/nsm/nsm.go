package nsm

import (
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"

	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"

	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
)

// ClientConnection is an interface for client connection
type ClientConnection interface {
	GetID() string
	GetConnectionSource() connection.Connection
	GetConnectionDestination() connection.Connection
	GetNetworkService() string
}

// NetworkServiceClient is an interface for network service client
type NetworkServiceClient interface {
	Request(ctx context.Context, request networkservice.Request) (connection.Connection, error)
	Close(ctx context.Context, connection connection.Connection) error

	Cleanup() error
}

// HealState - keep the cause of healing process
type HealState int32

const (
	// HealStateDstDown is a case when destination is down: we need to restore it and re-program local Dataplane.
	HealStateDstDown HealState = 1
	// HealStateSrcDown is a case when source is down: most probable will not happen yet.
	HealStateSrcDown HealState = 2
	// HealStateDataplaneDown is a case when local Dataplane is down: we need to heal NSE/Remote NSM and local Dataplane.
	HealStateDataplaneDown HealState = 3
	// HealStateDstUpdate is a case when destination is updated: we need to re-program local Dataplane.
	HealStateDstUpdate HealState = 4
	// HealStateDstNmgrDown is a case when destination and/or Remote NSM is down: we need to heal NSE/Remote NSM.
	HealStateDstNmgrDown HealState = 5
)

// NetworkServiceRequestManager - allow to provide local and remote service interfaces.
type NetworkServiceRequestManager interface {
	LocalManager() local_networkservice.NetworkServiceServer
	RemoteManager() remote_networkservice.NetworkServiceServer
}
type NetworkServiceManager interface {
	RestoreConnections(xcons []*crossconnect.CrossConnect, dataplane string)
	GetHealProperties() *Properties
	WaitForDataplane(ctx context.Context, duration time.Duration) error
	RemoteConnectionLost(ctx context.Context, clientConnection ClientConnection)
	NotifyRenamedEndpoint(nseOldName, nseNewName string)
	// Getters
	ServiceRegistry() serviceregistry.ServiceRegistry
	NseManager() NetworkServiceEndpointManager
	PluginRegistry() plugins.PluginRegistry
	Model() model.Model

	SetRemoteServer(server remote_networkservice.NetworkServiceServer)

	CloseConnection(ctx context.Context, clientConnection ClientConnection) error
	Heal(clientConnection ClientConnection, state HealState)
}

//NetworkServiceEndpointManager - manages endpoints, TODO: Will be removed in next PRs.
type NetworkServiceEndpointManager interface {
	GetEndpoint(ctx context.Context, requestConnection connection.Connection, ignoreEndpoints map[string]*registry.NSERegistration) (*registry.NSERegistration, error)
	CreateNSEClient(ctx context.Context, endpoint *registry.NSERegistration) (NetworkServiceClient, error)
	IsLocalEndpoint(endpoint *registry.NSERegistration) bool
	CheckUpdateNSE(ctx context.Context, reg *registry.NSERegistration) bool
}
