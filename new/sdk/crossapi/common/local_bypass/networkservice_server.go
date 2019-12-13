package local_bypass

import (
	"context"
	"net/url"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/client_url"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type localBypassServer struct {
	// Map of names -> *url.URLs for local bypass to file sockets
	sockets sync.Map
}

func NewServer(server *registry.NetworkServiceRegistryServer) networkservice.NetworkServiceServer {
	rv := &localBypassServer{}
	*server = newRegistryServer(rv)
	return rv
}

func (l *localBypassServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if v, ok := l.sockets.Load(request.GetConnection().GetNetworkServiceEndpointName()); ok && v != nil {
		if u, ok := v.(*url.URL); ok {
			ctx = client_url.WithClientUrl(ctx, u)
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (l *localBypassServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if v, ok := l.sockets.Load(conn.GetNetworkServiceEndpointName()); ok && v != nil {
		if u, ok := v.(*url.URL); ok {
			ctx = client_url.WithClientUrl(ctx, u)
		}
	}
	return next.Server(ctx).Close(ctx, conn)
}
