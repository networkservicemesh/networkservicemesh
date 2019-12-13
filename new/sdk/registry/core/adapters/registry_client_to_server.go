package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type registryClientToServer struct {
	client registry.NetworkServiceRegistryClient
}

func NewRegistryClientToServer(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryServer {
	return &registryClientToServer{client: client}
}

func (r *registryClientToServer) RegisterNSE(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
	return r.client.RegisterNSE(ctx, registration)
}

func (r *registryClientToServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	return r.client.RemoveNSE(ctx, request)
}
