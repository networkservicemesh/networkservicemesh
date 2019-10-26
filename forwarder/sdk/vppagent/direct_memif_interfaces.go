package forwarder

import (
	"context"

	memiIf "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	"github.com/networkservicemesh/networkservicemesh/forwarder/vppagent/pkg/memif"
)

//DirectMemifInterfaces creates forwarder server handler with direct memif connection/ disconnection
func DirectMemifInterfaces(baseDir string) forwarder.ForwarderServer {
	return &directMemifInterface{
		directMemifConnector: memif.NewDirectMemifConnector(baseDir),
	}
}

type directMemifInterface struct {
	directMemifConnector *memif.DirectMemifConnector
}

func (c *directMemifInterface) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	if isDirectMemif(crossConnect) {
		return c.directMemifConnector.Connect(crossConnect)
	}
	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	return next.Request(ctx, crossConnect)
}

func (c *directMemifInterface) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	if isDirectMemif(crossConnect) {
		c.directMemifConnector.Disconnect(crossConnect)
		return new(empty.Empty), nil
	}
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConnect)
}

func isDirectMemif(crossConnect *crossconnect.CrossConnect) bool {
	return crossConnect.GetLocalSource().GetMechanism().GetType() == memiIf.MECHANISM &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == memiIf.MECHANISM
}
