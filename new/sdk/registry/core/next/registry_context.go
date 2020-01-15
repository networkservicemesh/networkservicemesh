package next

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

const (
	nextRegistryServerKey contextKeyType = "NextRegistryServer"
	nextRegistryClientKey contextKeyType = "NextRegistryClient"
)

// withNextRegistryServer -
//    Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceRegistryServer to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextRegistryServer(parent context.Context, next registry.NetworkServiceRegistryServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextRegistryServerKey, next)
}

// RegistryServer -
//   Returns the RegistryServer registry.NetworkServiceRegistryServer to be called in the chain from the context.Context
func RegistryServer(ctx context.Context) registry.NetworkServiceRegistryServer {
	if rv, ok := ctx.Value(nextRegistryServerKey).(registry.NetworkServiceRegistryServer); ok {
		return rv
	}
	return nil
}

// withNextRegistryClient -
//    Wraps 'parent' in a new Context that has the RegistryClient registry.NetworkServiceRegistryClient to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextRegistryClient(parent context.Context, next registry.NetworkServiceRegistryClient) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextRegistryClientKey, next)
}

// RegistryClient -
//   Returns the RegistryClient registry.NetworkServiceRegistryClient to be called in the chain from the context.Context
func RegistryClient(ctx context.Context) registry.NetworkServiceRegistryClient {
	if rv, ok := ctx.Value(nextDiscoveryClientKey).(registry.NetworkServiceRegistryClient); ok {
		return rv
	}
	return nil
}
