package nsm

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"golang.org/x/net/context"
	"time"
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
	GetId() string
	GetConnectionSource() NSMConnection
	GetNetworkService() string
}

type NetworkServiceClient interface {
	Request(ctx context.Context, request NSMRequest) (NSMConnection, error)
	Close(ctx context.Context, connection NSMConnection) error

	Cleanup() error
}

type HealState int32

const (
	HealState_DstDown             HealState = 1 // Destination is down, we need to restore it and re-program local Datplane.
	HealState_SrcDown             HealState = 2 // Source is down, most probable will not happen yet.
	HealState_DataplaneDown       HealState = 3 // In case local Dataplane is down, we need to heal NSE/Remote NSM and Dataplane.
	HealState_RemoteDataplaneDown HealState = 4 // Remote Dataplane is down, we need to re-program local dataplane.
	HealState_DstNmgrDown         HealState = 5 // Destination is updated, most probable because of Remote Dataplane is down, we need to re-program local dataplane.
)

type NetworkServiceManager interface {
	Request(ctx context.Context, request NSMRequest) (NSMConnection, error)
	Close(ctx context.Context, clientConnection NSMClientConnection) error
	Heal(connection NSMClientConnection, healState HealState)
	RestoreConnections(xcons []*crossconnect.CrossConnect, dataplane string)
	GetHealProperties() *NsmProperties
	WaitForDataplane(duration time.Duration) error
	RemoteConnectionLost(clientConnection NSMClientConnection)
}
