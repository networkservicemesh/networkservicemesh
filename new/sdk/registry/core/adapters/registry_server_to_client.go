package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"google.golang.org/grpc"
)

type registryServerToClient struct {
	server registry.NetworkServiceRegistryServer
}

func NewRegistryServerToClient(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryClient {
	return &registryServerToClient{server: server}
}

func (r *registryServerToClient) RegisterNSE(ctx context.Context, registration *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	return r.server.RegisterNSE(ctx, registration)
}

func (r *registryServerToClient) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return r.server.RemoveNSE(ctx, request)
}
