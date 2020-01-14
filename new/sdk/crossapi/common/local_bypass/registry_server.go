package local_bypass

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/next"
)

type localRegistry struct {
	server *localBypassServer
}

func newRegistryServer(server *localBypassServer) registry.NetworkServiceRegistryServer {
	return &localRegistry{server: server}
}

func (n *localRegistry) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	p, ok := peer.FromContext(ctx)
	if ok && p.Addr.Network() == "unix" && n.server != nil {
		u := &url.URL{
			Scheme: "unix",
			Path:   p.Addr.String(),
		}
		n.server.sockets.LoadOrStore(request.GetNetworkServiceEndpoint(), u)
	}
	return next.RegistryServer(ctx).RegisterNSE(ctx, request)
}

func (n *localRegistry) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	if n.server != nil {
		n.server.sockets.Delete(request.GetNetworkServiceEndpointName())
	}
	return next.RegistryServer(ctx).RemoveNSE(ctx, request)
}
