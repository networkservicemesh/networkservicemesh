package adapters

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type discoveryClientToServer struct {
	client registry.NetworkServiceDiscoveryClient
}

func NewDiscoveryClientToServer(client registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryServer {
	return &discoveryClientToServer{client: client}
}

func (c *discoveryClientToServer) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	return c.client.FindNetworkService(ctx, request)
}
