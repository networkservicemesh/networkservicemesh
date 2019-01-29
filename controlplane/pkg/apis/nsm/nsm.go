package nsm

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"golang.org/x/net/context"
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
	Unitifed Connection interface, handles common part of local/Remote connections.
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
	GetSourceConnection() NSMConnection
	GetNetworkService() string
}

type NetworkServiceClient interface {
	Request(ctx context.Context, request NSMRequest) (NSMConnection, error)
	Close(ctx context.Context, connection NSMConnection) error

	Cleanup() error
}

type HealState int32

const (
	HealState_DstDown       HealState = 1
	HealState_SrcDown       HealState = 2
	HealState_DataplaneDown HealState = 3
)

type NetworkServiceManager interface {
	Request(ctx context.Context, request NSMRequest) (NSMConnection, error)
	Close(ctx context.Context, clientConnection NSMClientConnection) error
	Heal(connection NSMClientConnection, healState HealState)
}
