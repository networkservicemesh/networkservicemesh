package chain

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/next"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/trace"
)

func NewNetworkServiceRegistryServer(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return next.NewWrappedNetworkServiceRegistryServer(trace.NewNetworkServiceRegistryServer, servers...)
}
