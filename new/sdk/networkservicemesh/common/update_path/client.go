package update_path

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type updatePathClient struct {
	name string
}

func NewClient(name string) networkservice.NetworkServiceClient {
	return &updatePathClient{name: name}
}

func (u *updatePathClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	// Handle zero index case
	if int(request.GetConnection().GetPath().GetIndex()) == len(request.GetConnection().GetPath().GetPathSegments()) {
		request.GetConnection().GetPath().PathSegments = append(request.GetConnection().GetPath().PathSegments, &connection.PathSegment{})
	}
	if int(request.GetConnection().GetPath().GetIndex()) >= len(request.GetConnection().GetPath().GetPathSegments()) {
		return nil, errors.Errorf("NetworkServiceRequest.Connection.Path.Index(%d) >= len(NetworkServiceRequest.Connection.Path.PathSegments)(%d)",
			request.GetConnection().GetPath().GetIndex(),
			len(request.GetConnection().GetPath().GetPathSegments()))
	}
	request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Name = u.name
	request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Id = request.GetConnection().GetId()
	// TODO set token and expiration
	// request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Token =
	// request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Expires =
	return next.Client(ctx).Request(ctx, request)
}

func (u *updatePathClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn)
}
