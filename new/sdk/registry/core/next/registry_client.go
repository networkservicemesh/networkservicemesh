package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"google.golang.org/grpc"
)

type RegistryClientWrapper func(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient
type RegistryClientChainer func(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient

type nextRegistryClient struct {
	index   int
	clients []registry.NetworkServiceRegistryClient
}

func NewWrappedNetworkServiceRegistryClient(wrapper RegistryClientWrapper, clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	rv := &nextRegistryClient{
		clients: clients,
	}
	for i := range rv.clients {
		rv.clients[i] = wrapper(rv.clients[i])
	}
	return rv
}

func NewNetworkServiceRegistryClient(clients []registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return NewWrappedNetworkServiceRegistryClient(nil, clients...)
}

func (n *nextRegistryClient) RegisterNSE(ctx context.Context, request *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].RegisterNSE(withNextRegistryClient(ctx, &nextRegistryClient{clients: n.clients, index: n.index + 1}), request)
	}
	return n.clients[n.index].RegisterNSE(withNextRegistryClient(ctx, nil), request, opts...)
}

func (n *nextRegistryClient) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].RemoveNSE(withNextRegistryClient(ctx, &nextRegistryClient{clients: n.clients, index: n.index + 1}), request)
	}
	return n.clients[n.index].RemoveNSE(withNextRegistryClient(ctx, nil), request, opts...)
}
