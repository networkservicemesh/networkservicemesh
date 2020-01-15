package next

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

const (
	nextServerKey contextKeyType = "NextServer"
	nextClientKey contextKeyType = "NextClient"
)

type contextKeyType string

// withNextServer -
//    Wraps 'parent' in a new Context that has the Server networkservice.NetworkServiceServer to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextServer(parent context.Context, next networkservice.NetworkServiceServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextServerKey, next)
}

// Server -
//   Returns the Server networkservice.NetworkServiceServer to be called in the chain from the context.Context
func Server(ctx context.Context) networkservice.NetworkServiceServer {
	if rv, ok := ctx.Value(nextServerKey).(networkservice.NetworkServiceServer); ok {
		return rv
	}
	return nil
}

// withNextClient -
//    Wraps 'parent' in a new Context that has the Server networkservice.NetworkServiceServer to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNextClient(parent context.Context, next networkservice.NetworkServiceClient) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextClientKey, next)
}

// Client -
//   Returns the Client networkservice.NetworkServiceClient to be called in the chain from the context.Context
func Client(ctx context.Context) networkservice.NetworkServiceClient {
	if rv, ok := ctx.Value(nextClientKey).(networkservice.NetworkServiceClient); ok {
		return rv
	}
	return nil
}
