package endpoint

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

type RoutesConfiguration []string

type RoutesEndpoint struct {
	routes []*connectioncontext.Route
}

func (r *RoutesEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	request.GetConnection().GetContext().Routes = r.routes
	if Next(ctx) != nil {
		return Next(ctx).Request(ctx, request)
	}
	return nil, nil
}

func (r *RoutesEndpoint) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, conn)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (r *RoutesEndpoint) Name() string {
	return "routes"
}

func NewRoutesEndpoint(prefixes []string) *RoutesEndpoint {
	var routes []*connectioncontext.Route
	for _, prefix := range prefixes {
		routes = append(routes, &connectioncontext.Route{Prefix: prefix})
	}
	return &RoutesEndpoint{
		routes: routes,
	}
}
