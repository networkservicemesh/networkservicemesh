package setid

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type idServer struct {
	name string
}

func NewServer(name string) networkservice.NetworkServiceServer {
	return &idServer{name: name}
}

func (i *idServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if len(request.GetConnection().GetPath().GetPathSegments()) > int(request.GetConnection().GetPath().GetIndex()) &&
		request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].GetName() != i.name &&
		request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].GetId() != request.GetConnection().GetId() {
		request.GetConnection().Id = uuid.New().String()
	}
	return next.Server(ctx).Request(ctx, request)
}

func (i *idServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
