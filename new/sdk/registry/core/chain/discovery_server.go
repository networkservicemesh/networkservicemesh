package chain

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/next"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/registry/core/trace"
)

func NewNetworkServiceDiscoveryServer(servers ...registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer {
	return next.NewWrappedNetworkServiceDiscoveryServer(trace.NewNetworkServiceDiscoveryServer, servers...)
}
