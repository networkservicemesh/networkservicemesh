package dataplane

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent/dataplane/state"
)

func Connect(endpoint string) dataplane.DataplaneServer {
	return &connect{endpoint: endpoint}
}

type connect struct {
	*EmptyChainedDataplaneServer
	endpoint string
}

func (c *connect) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	nextCtx, err := state.WithConfiguratorClient(ctx, c.endpoint)
	if err != nil {
		return nil, err
	}
	return state.NextDataplaneRequest(nextCtx, crossConnect)
}

func (c *connect) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	nextCtx, err := state.WithConfiguratorClient(ctx, c.endpoint)
	if err != nil {
		return nil, err
	}
	return state.NextDataplaneClose(nextCtx, crossConnect)
}
