package chain

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/next"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/trace"
)

func NewNetworkServiceDiscoveryClient(clients ...registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient {
	return next.NewWrappedNetworkServiceDiscoveryClient(trace.NewNetworkServiceDiscoveryClient, clients...)
}
