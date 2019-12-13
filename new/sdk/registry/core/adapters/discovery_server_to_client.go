package adapters

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"google.golang.org/grpc"
)

type discoveryServerToClient struct {
	server registry.NetworkServiceDiscoveryServer
}

func NewDiscoveryServerToClient(server registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryClient {
	return &discoveryServerToClient{server: server}
}

func (s *discoveryServerToClient) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest, opts ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	return s.server.FindNetworkService(ctx, request)
}
