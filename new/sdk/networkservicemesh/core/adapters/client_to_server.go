package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

type clientToServer struct {
	client networkservice.NetworkServiceClient
}

func NewClientToServer(client networkservice.NetworkServiceClient) networkservice.NetworkServiceServer {
	return &clientToServer{client: client}
}

func (c *clientToServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	return c.Request(ctx, request)
}

func (c *clientToServer) Close(ctx context.Context, request *connection.Connection) (*empty.Empty, error) {
	return c.Close(ctx, request)
}
