package nsmgr

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/crossapi/chains/client"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/crossapi/chains/endpoint"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/crossapi/common/local_bypass"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/connect"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/discover_candidates"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/round_robin"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/adapters"
	adapter_registry "github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/adapters"
	chain_registry "github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/chain"
	"google.golang.org/grpc"
)

type Nsmgr interface {
	endpoint.Endpoint
	registry.NetworkServiceRegistryServer
	registry.NetworkServiceDiscoveryServer
}

type nsmgr struct {
	endpoint.Endpoint
	registry.NetworkServiceRegistryServer
	registry.NetworkServiceDiscoveryServer
}

func NewNsmgr(name string, registryCC *grpc.ClientConn) Nsmgr {
	rv := &nsmgr{}
	rv.Endpoint = endpoint.NewServer(
		name,
		discover_candidates.NewServer(registry.NewNetworkServiceDiscoveryClient(registryCC)),
		round_robin.NewServer(),
		local_bypass.NewServer(&rv.NetworkServiceRegistryServer),
		connect.NewServer(client.NewClientFactory(name, adapters.NewServerToClient(rv))),
	)
	rv.NetworkServiceRegistryServer = chain_registry.NewNetworkServiceRegistryServer(
		rv.NetworkServiceRegistryServer,
		adapter_registry.NewRegistryClientToServer(registry.NewNetworkServiceRegistryClient(registryCC)),
	)
	rv.NetworkServiceDiscoveryServer = adapter_registry.NewDiscoveryClientToServer(registry.NewNetworkServiceDiscoveryClient(registryCC))
	return rv
}

func (n *nsmgr) Register(s *grpc.Server) {
	n.Endpoint.Register(s)
	registry.RegisterNetworkServiceRegistryServer(s, n)
	registry.RegisterNetworkServiceDiscoveryServer(s, n)
}
