package nsm

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"golang.org/x/net/context"
)

type NSMRequest interface {
	IsValid() error
	IsRemote() bool
}
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
	Request(ctx context.Context, request NSMRequest, extra_parameters map[string]string) (NSMConnection, error)
	Close(ctx context.Context, clientConnection *model.ClientConnection) error
	Heal(connection *model.ClientConnection, healState HealState)
}
