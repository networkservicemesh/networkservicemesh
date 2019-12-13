package next

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type DiscoveryServerWrapper func(server registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer
type DiscoveryServerChainer func(servers ...registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer

type nextDiscoveryServer struct {
	index   int
	servers []registry.NetworkServiceDiscoveryServer
}

func NewWrappedNetworkServiceDiscoveryServer(wrapper DiscoveryServerWrapper, servers ...registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer {
	rv := &nextDiscoveryServer{
		servers: servers,
	}
	for i := range rv.servers {
		rv.servers[i] = wrapper(rv.servers[i])
	}
	return rv
}

func NewNetworkServiceDiscoveryServer(servers []registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer {
	return NewWrappedNetworkServiceDiscoveryServer(nil, servers...)
}

func (n *nextDiscoveryServer) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].FindNetworkService(withNextDiscoveryServer(ctx, &nextDiscoveryServer{servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].FindNetworkService(withNextDiscoveryServer(ctx, nil), request)
}
