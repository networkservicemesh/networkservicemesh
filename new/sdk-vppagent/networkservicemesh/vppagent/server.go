package vppagent

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type configServer struct{}

func NewServer() networkservice.NetworkServiceServer {
	return &configServer{}
}

func (c *configServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	return next.Server(ctx).Request(withConfig(ctx), request)
}

func (c *configServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(withConfig(ctx), conn)
}
