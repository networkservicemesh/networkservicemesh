package authorize

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type authorizeClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &authorizeClient{}
}

func (a *authorizeClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	// TODO implement authorization
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (a *authorizeClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// TODO implement authorization
	return next.Client(ctx).Close(ctx, conn, opts...)
}
