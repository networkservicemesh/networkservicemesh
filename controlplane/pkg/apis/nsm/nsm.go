package nsm

import (
	"time"

	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
)

// ClientConnection is an interface for client connection
type ClientConnection interface {
	GetID() string
	GetConnectionSource() connection.Connection
	GetConnectionDestination() connection.Connection
	GetNetworkService() string
}

type NetworkServiceClient interface {
	Request(ctx context.Context, request networkservice.Request) (connection.Connection, error)
	Close(ctx context.Context, connection connection.Connection) error

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
	Request(ctx context.Context, request networkservice.Request) (connection.Connection, error)
	Close(ctx context.Context, clientConnection ClientConnection) error
	Heal(clientConnection ClientConnection, healState HealState)
	RestoreConnections(xcons []*crossconnect.CrossConnect, dataplane string)
	GetHealProperties() *NsmProperties
	WaitForDataplane(duration time.Duration) error
	RemoteConnectionLost(clientConnection ClientConnection)
	NotifyRenamedEndpoint(nseOldName, nseNewName string)
}
