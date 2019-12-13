package copy_client_connection_context

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/connect"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type copyClientConnectionContext struct{}

func NewServer() networkservice.NetworkServiceServer {
	return &copyClientConnectionContext{}
}

func (c *copyClientConnectionContext) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	request.GetConnection().Context = connect.ClientConnection(ctx).GetContext()
	return next.Server(ctx).Request(ctx, request)
}

func (c *copyClientConnectionContext) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	conn.Context = connect.ClientConnection(ctx).GetContext()
	return next.Server(ctx).Close(ctx, conn)
}
