//Package vppagent provides sdk API for creating chaining vppagent forwarders
package vppagent

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

type chainedForwarderServer struct {
	handlers []forwarder.ForwarderServer
}

func (c *chainedForwarderServer) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	span := spanhelper.FromContext(ctx, "Forwarder.Request")
	defer span.Finish()
	span.LogObject("crossConnect", crossConnect)

	if len(c.handlers) == 0 {
		logrus.Info("chainedForwarderServer: has not handlers for next request")
		return crossConnect, nil
	}
	next := &next{handlers: c.handlers, index: 0}
	nextCtx := withNext(span.Context(), next)
	return next.Request(nextCtx, crossConnect)
}

func (c *chainedForwarderServer) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	span := spanhelper.FromContext(ctx, "Forwarder.Close")
	span.LogObject("crossConnect", crossConnect)
	defer span.Finish()
	if len(c.handlers) == 0 {
		logrus.Info("chainedForwarderServer: has not handlers for next close")
		return new(empty.Empty), nil
	}
	next := &next{handlers: c.handlers, index: 0}
	nextCtx := withNext(span.Context(), next)
	return next.Close(nextCtx, crossConnect)
}

// ChainOf makes chain of forwarder server handlers
func ChainOf(handlers ...forwarder.ForwarderServer) forwarder.ForwarderServer {
	return &chainedForwarderServer{handlers: handlers}
}
