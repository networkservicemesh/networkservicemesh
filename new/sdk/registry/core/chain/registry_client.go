package chain

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/next"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/trace"
)

func NewNetworkServiceRegistryClient(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return next.NewWrappedNetworkServiceRegistryClient(trace.NewNetworkServiceRegistryClient, clients...)
}
