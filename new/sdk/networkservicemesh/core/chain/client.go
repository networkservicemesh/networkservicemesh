package chain

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/trace"
)

func NewNetworkServiceClient(clients ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return next.NewWrappedNetworkServiceClient(trace.NewNetworkServiceClient, clients...)
}
