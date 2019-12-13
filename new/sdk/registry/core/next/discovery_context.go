package next

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

const (
	nextDiscoveryServerKey contextKeyType = "NextDiscoveryServer"
	nextDiscoveryClientKey contextKeyType = "NextDiscoveryClient"
)

type contextKeyType string

// withNextDiscoveryServer -
//    Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceDiscoveryServer to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextDiscoveryServer(parent context.Context, next registry.NetworkServiceDiscoveryServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextDiscoveryServerKey, next)
}

// DiscoveryServer -
//   Returns the DiscoveryServer networkservice.NetworkServiceServer to be called in the chain from the context.Context
func DiscoveryServer(ctx context.Context) registry.NetworkServiceDiscoveryServer {
	if rv, ok := ctx.Value(nextDiscoveryServerKey).(registry.NetworkServiceDiscoveryServer); ok {
		return rv
	}
	return nil
}

// withNextDiscoveryClient -
//    Wraps 'parent' in a new Context that has the DiscoveryServer registry.NetworkServiceDiscoveryClient to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextDiscoveryClient(parent context.Context, next registry.NetworkServiceDiscoveryClient) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextDiscoveryClientKey, next)
}

// DiscoveryClient -
//   Returns the DiscoveryClient registry.NetworkServiceDiscoveryClient to be called in the chain from the context.Context
func DiscoveryClient(ctx context.Context) registry.NetworkServiceDiscoveryClient {
	if rv, ok := ctx.Value(nextDiscoveryClientKey).(registry.NetworkServiceDiscoveryClient); ok {
		return rv
	}
	return nil
}
