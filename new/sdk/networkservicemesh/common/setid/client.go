package setid

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type idClient struct {
	name string
}

// NewClient - creates a new setId client.
//             name - name of the client
//             Iff the current pathSegment name != name && pathsegment.id != connection.Id, set a new uuid for
//             connection id
func NewClient(name string) networkservice.NetworkServiceClient {
	return &idClient{name: name}
}

func (i *idClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	if len(request.GetConnection().GetPath().GetPathSegments()) > int(request.GetConnection().GetPath().GetIndex()) &&
		request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].GetName() != i.name &&
		request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].GetId() != request.GetConnection().GetId() {
		request.GetConnection().Id = uuid.New().String()
	}
	return next.Client(ctx).Request(ctx, request)
}

func (i *idClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
