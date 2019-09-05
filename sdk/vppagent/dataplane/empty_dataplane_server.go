package dataplane

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent/dataplane/state"
)

type EmptyChainedDataplaneServer struct {
}

func (c *EmptyChainedDataplaneServer) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	return state.NextDataplaneRequest(ctx, crossConnect)
}

func (c *EmptyChainedDataplaneServer) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	return state.NextDataplaneClose(ctx, crossConnect)
}

func (s *EmptyChainedDataplaneServer) MonitorMechanisms(empty *empty.Empty, monitorServer dataplane.Dataplane_MonitorMechanismsServer) error {
	return nil
}
