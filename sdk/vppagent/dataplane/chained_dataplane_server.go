package dataplane

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
)

type chainedDataplaneServer struct {
	handlers []dataplane.DataplaneServer
}

func (c *chainedDataplaneServer) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {

	if len(c.handlers) == 0 {
		logrus.Info("chainedDataplaneServer: has not handlers for next request")
		return crossConnect, nil
	}

	nextCtx := WithChain(ctx, c, c.handlers)
	return c.handlers[0].Request(nextCtx, crossConnect)

}

func (c *chainedDataplaneServer) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	if len(c.handlers) == 0 {
		logrus.Info("chainedDataplaneServer: has not handlers for next close")
		return new(empty.Empty), nil
	}
	nextCtx := WithChain(ctx, c, c.handlers)
	return c.handlers[0].Close(nextCtx, crossConnect)
}

// ChainOf makes chain of dataplane server handlers
func ChainOf(handlers ...dataplane.DataplaneServer) dataplane.DataplaneServer {
	return &chainedDataplaneServer{handlers: handlers}
}
