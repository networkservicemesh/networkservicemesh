package update_path

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/pkg/errors"
)

type updatePathServer struct {
	name string
}

func NewServer(name string) *updatePathServer {
	return &updatePathServer{name: name}
}

func (u *updatePathServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if int(request.GetConnection().GetPath().GetIndex()) >= len(request.GetConnection().GetPath().GetPathSegments()) {
		return nil, errors.Errorf("NetworkServiceRequest.Connection.Path.Index(%d) >= len(NetworkServiceRequest.Connection.Path.PathSegments)(%d)",
			request.GetConnection().GetPath().GetIndex(),
			len(request.GetConnection().GetPath().GetPathSegments()))
	}
	// increment the index
	request.GetConnection().GetPath().Index++
	// extend the path (presuming that we need to)
	if int(request.GetConnection().GetPath().GetIndex()) == len(request.GetConnection().GetPath().GetPathSegments()) {
		request.GetConnection().GetPath().PathSegments = append(request.GetConnection().GetPath().PathSegments, &connection.PathSegment{})
	}
	request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Name = u.name
	request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Id = request.GetConnection().GetId()
	// TODO set token and expiration
	// request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Token =
	// request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].Expires =
	return next.Server(ctx).Request(ctx, request)
}

func (u *updatePathServer) Close(context.Context, *connection.Connection) (*empty.Empty, error) {
	panic("implement me")
}
