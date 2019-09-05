package dataplane

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent/dataplane/state"
)

type ChainedDataplaneServer interface {
	dataplane.DataplaneServer
}

type chainedDataplaneServer struct {
	handlers []dataplane.DataplaneServer
}

func (c *chainedDataplaneServer) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	nextCtx := state.WithChain(ctx, c, c.handlers)
	return state.NextDataplaneRequest(nextCtx, crossConnect)
}

func (c *chainedDataplaneServer) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	nextCtx := state.WithChain(ctx, c, c.handlers)
	return state.NextDataplaneClose(nextCtx, crossConnect)
}

//TODO: do not support MonitorMechanisms for chain
func (c *chainedDataplaneServer) MonitorMechanisms(empty *empty.Empty, monitorServer dataplane.Dataplane_MonitorMechanismsServer) error {
	ctx := state.WithChain(context.Background(), c, c.handlers)
	current := state.NextDataplaneServer(ctx)
	var err error
	for current != nil && err == nil {
		err = current.MonitorMechanisms(empty, monitorServer)
		current = state.NextDataplaneServer(ctx)
	}
	return err
}

func ChainOf(handlers ...dataplane.DataplaneServer) dataplane.DataplaneServer {
	return &chainedDataplaneServer{handlers: handlers}
}
