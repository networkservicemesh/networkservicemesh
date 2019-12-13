package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"google.golang.org/grpc"
)

type serverToClient struct {
	server networkservice.NetworkServiceServer
}

func NewServerToClient(server networkservice.NetworkServiceServer) networkservice.NetworkServiceClient {
	return &serverToClient{server: server}
}

func (s *serverToClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	return s.server.Request(ctx, in)
}

func (s *serverToClient) Close(ctx context.Context, in *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return s.server.Close(ctx, in)
}
