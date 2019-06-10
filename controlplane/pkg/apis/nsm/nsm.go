package nsm

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"
	"time"

	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
)

/*
	Unified request, handles common part of local/Remote network requests.
*/
type NSMRequest interface {
	IsValid() error
	IsRemote() bool
	GetConnectionId() string
	Clone() NSMRequest
	SetConnection(connection NSMConnection)
}

/*
	Unified Connection interface, handles common part of local/Remote connections.
*/
type NSMConnection interface {
	IsValid() error
	SetId(id string)
	GetNetworkService() string
	GetContext() *connectioncontext.ConnectionContext
	UpdateContext(connectionContext *connectioncontext.ConnectionContext) error
	SetContext(connectionContext *connectioncontext.ConnectionContext)
	GetId() string
	IsComplete() error
	GetLabels() map[string]string
	GetNetworkServiceEndpointName() string
	SetNetworkServiceName(service string)
}

type NSMClientConnection interface {
	GetID() string
	GetConnectionSource() NSMConnection
	GetConnectionDestination() NSMConnection
	GetNetworkService() string
}

type NetworkServiceClient interface {
	Request(ctx context.Context, request NSMRequest) (NSMConnection, error)
	Close(ctx context.Context, connection NSMConnection) error

	Cleanup() error
}

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

type NetworkServiceManager interface {
	Request(ctx context.Context, request NSMRequest) (NSMConnection, error)
	Close(ctx context.Context, clientConnection NSMClientConnection) error
	Heal(connection NSMClientConnection, healState HealState)
	RestoreConnections(xcons []*crossconnect.CrossConnect, dataplane string)
	GetHealProperties() *NsmProperties
	WaitForDataplane(duration time.Duration) error
	RemoteConnectionLost(clientConnection NSMClientConnection)
	GetExcludePrefixes() prefix_pool.PrefixPool
	SetExcludePrefixes(prefix_pool.PrefixPool)
}
