package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

// tailServer is a simple implementation of networkservice.NetworkServiceServer that is called at the end of a chain
// to insure that we never call a method on a nil object
type tailServer struct{}

func newTailServer() *tailServer {
	return &tailServer{}
}

func (t *tailServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	return request.GetConnection(), nil
}

func (t *tailServer) Close(context.Context, *connection.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
