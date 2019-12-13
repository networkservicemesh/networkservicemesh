package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type RegistryServerWrapper func(client registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer
type RegistryServerChainer func(clients ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer

type nextRegistryServer struct {
	index   int
	servers []registry.NetworkServiceRegistryServer
}

func NewWrappedNetworkServiceRegistryServer(wrapper RegistryServerWrapper, servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	rv := &nextRegistryServer{
		servers: servers,
	}
	for i := range rv.servers {
		rv.servers[i] = wrapper(rv.servers[i])
	}
	return rv
}

func NewNetworkServiceRegistryServer(clients []registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return NewWrappedNetworkServiceRegistryClient(nil, clients...)
}

func (n *nextRegistryServer) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].RegisterNSE(withNextRegistryServer(ctx, &nextRegistryServer{servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].RegisterNSE(withNextRegistryServer(ctx, nil), request)
}

func (n *nextRegistryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].RemoveNSE(withNextRegistryServer(ctx, &nextRegistryServer{servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].RemoveNSE(withNextRegistryServer(ctx, nil), request)
}
