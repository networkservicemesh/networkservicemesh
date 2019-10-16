package forwarder

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

//Connect creates forwarder server handler with connection to vpp-agent confgirator server
func Connect(endpoint string) forwarder.DataplaneServer {
	return &connect{endpoint: endpoint}
}

type connect struct {
	endpoint string
}

func (c *connect) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	nextCtx, close, err := WithConfiguratorClient(ctx, c.endpoint)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := close()
		if err != nil {
			Logger(ctx).Errorf("An error during closing configuration client: %v", err)
		}
	}()
	if next := Next(ctx); next != nil {
		return next.Request(nextCtx, crossConnect)
	}
	return crossConnect, nil
}

func (c *connect) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	nextCtx, close, err := WithConfiguratorClient(ctx, c.endpoint)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := close()
		if err != nil {
			Logger(ctx).Errorf("An error during closing configuration client: %v", err)
		}
	}()
	if next := Next(ctx); next != nil {
		_, err = next.Close(nextCtx, crossConnect)
		if err != nil {
			Logger(ctx).Errorf("An error during closing connection: %v", err)
		}
		return new(empty.Empty), nil
	}
	return new(empty.Empty), nil
}
