package authorize

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type authorizeServer struct{}

func NewServer() networkservice.NetworkServiceServer {
	return &authorizeServer{}
}

func (a *authorizeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	// TODO check authorization
	return next.Server(ctx).Request(ctx, request)
}

func (a *authorizeServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	// TODO check authorization
	return next.Server(ctx).Close(ctx, conn)
}
