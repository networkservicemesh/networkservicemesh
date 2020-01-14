package client_url

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type clientUrl struct {
	u *url.URL
}

func NewServer(u *url.URL) networkservice.NetworkServiceServer {
	return &clientUrl{u: u}
}

func (c *clientUrl) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = WithClientUrl(ctx, c.u)
	return next.Server(ctx).Request(ctx, request)
}

func (c *clientUrl) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	ctx = WithClientUrl(ctx, c.u)
	return next.Server(ctx).Close(ctx, conn)
}
