package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"google.golang.org/grpc"
)

// tailServer is a simple implementation of networkservice.NetworkServiceServer that is called at the end of a chain
// to insure that we never call a method on a nil object
type tailClient struct{}

func newTailClient() *tailClient {
	return &tailClient{}
}

func (t *tailClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	return request.GetConnection(), nil
}

func (t *tailClient) Close(context.Context, *connection.Connection, ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
