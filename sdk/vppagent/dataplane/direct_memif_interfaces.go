package dataplane

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/memif"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent/dataplane/state"
)

func DirectMemifInterfaces(baseDir string) dataplane.DataplaneServer {
	return &directMemifInterface{
		directMemifConnector:        memif.NewDirectMemifConnector(baseDir),
		EmptyChainedDataplaneServer: new(EmptyChainedDataplaneServer),
	}
}

type directMemifInterface struct {
	*EmptyChainedDataplaneServer
	directMemifConnector *memif.DirectMemifConnector
}

func (c *directMemifInterface) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	if isDirectMemif(crossConnect) {
		c.directMemifConnector.Connect(crossConnect)
	}
	return state.NextDataplaneRequest(ctx, crossConnect)
}

func (c *directMemifInterface) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	if isDirectMemif(crossConnect) {
		c.directMemifConnector.Disconnect(crossConnect)
		return new(empty.Empty), nil
	}
	return state.NextDataplaneClose(ctx, crossConnect)
}

func isDirectMemif(crossConnect *crossconnect.CrossConnect) bool {
	return crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_MEM_INTERFACE &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_MEM_INTERFACE
}
