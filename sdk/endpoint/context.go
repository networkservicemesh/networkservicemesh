package endpoint

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/local"
)

type contextKeyType string

const (
	clientConnectionKey contextKeyType = "ClientConnection"
	monitorServerKey    contextKeyType = "MonitorServer"
	nextKey             contextKeyType = "Next"
)

// WithClientConnection -
//   Wraps 'parent' in a new Context that has the ClientConnection
//   provided in:
//       connection *connection.Connection
//   using Context.Value(...) and returns the result.
//   Note: any previously existing ClientConnection will be overwritten.
//
//   Recommended use: in any Request or Close call that creates a ClientConnection, call:
//      ctx = WithClientConnection(ctx)
//   to ensure that the ctx has a ClientConnection
//   In any Request or Close call that consumes a ClientConnection, call:
//	    connectionMap := ClientConnection(ctx)
//   to retrieve the ClientConnection from the context.Context
func WithClientConnection(parent context.Context, connection *connection.Connection) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, clientConnectionKey, connection)
}

// ClientConnection -
//    Returns a ClientConnection from:
//      ctx context.Context
//    If any is present, otherwise nil
func ClientConnection(ctx context.Context) *connection.Connection {
	return ctx.Value(clientConnectionKey).(*connection.Connection)
}

// withMonitorServer -
//   Wraps 'parent' in a new Context that has the MonitorServer
//   provided in:
//       monitorServer local.MonitorServer
//   using Context.Value(...) and returns the result.
//   Note: should only be called from NSMEndpoint.Request/Close
//
func withMonitorServer(parent context.Context, monitorServer local.MonitorServer) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, monitorServerKey, monitorServer)
}

// MonitorServer -
//    Returns a MonitorServer from:
//      ctx context.Context
//    If any is present, otherwise nil
//    Should always be present if running via NSMEndpoint
func MonitorServer(ctx context.Context) local.MonitorServer {
	return ctx.Value(monitorServerKey).(local.MonitorServer)
}

// withNext -
//    Wraps 'parent' in a new Context that has the Next networkservice.NetworkServiceServer to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNext(parent context.Context, next networkservice.NetworkServiceServer) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, nextKey, next)
}

// Next -
//   Returns the Next networkservice.NetworkServiceServer to be called in the chain from the context.Context
func Next(ctx context.Context) networkservice.NetworkServiceServer {
	if rv, ok := ctx.Value(nextKey).(networkservice.NetworkServiceServer); ok {
		return rv
	}
	return nil
}
