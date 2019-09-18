package endpoint

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
)

// RoutesConfiguration is a list of prefixes for routes
type RoutesConfiguration []string

// RoutesEndpoint -
//   Adds routes to the ConnectionContext for the Request
type RoutesEndpoint struct {
	routes []*connectioncontext.Route
}

// Request handler
//  Consumes from ctx context.Context:
//    Next
func (r *RoutesEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	request.GetConnection().GetContext().GetIpContext().DstRoutes = r.routes
	if Next(ctx) != nil {
		return Next(ctx).Request(ctx, request)
	}
	return nil, nil
}

// Close handler
//   Consumes from ctx context.Context:
//     Next
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

// NewRoutesEndpoint creates New RoutesEndpoint
func NewRoutesEndpoint(prefixes []string) *RoutesEndpoint {
	routes := make([]*connectioncontext.Route, 1)
	for _, prefix := range prefixes {
		routes = append(routes, &connectioncontext.Route{Prefix: prefix})
	}
	return &RoutesEndpoint{
		routes: routes,
	}
}
