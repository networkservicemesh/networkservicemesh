package chain

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/trace"
)

func NewNetworkServiceServer(servers ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, servers...)
}
