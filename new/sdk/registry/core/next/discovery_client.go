package next

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type DiscoveryClientWrapper func(client registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient
type DiscoveryClientChainer func(clients ...registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient

type nextDiscoveryClient struct {
	index   int
	clients []registry.NetworkServiceDiscoveryClient
}

func NewWrappedNetworkServiceDiscoveryClient(wrapper DiscoveryClientWrapper, clients ...registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient {
	rv := &nextDiscoveryClient{
		clients: clients,
	}
	for i := range rv.clients {
		rv.clients[i] = wrapper(rv.clients[i])
	}
	return rv
}

func NewNetworkServiceDiscoveryClient(clients []registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient {
	return NewWrappedNetworkServiceDiscoveryClient(nil, clients...)
}

func (n *nextDiscoveryClient) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest, opts ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].FindNetworkService(withNextDiscoveryClient(ctx, &nextDiscoveryClient{clients: n.clients, index: n.index + 1}), request, opts...)
	}
	return n.clients[n.index].FindNetworkService(withNextDiscoveryClient(ctx, nil), request, opts...)
}
