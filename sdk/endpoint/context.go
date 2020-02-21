package endpoint

import (
	"context"

	connectionMonitor "github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type contextKeyType string

const (
	clientConnectionKey contextKeyType = "ClientConnection"
	monitorServerKey    contextKeyType = "MonitorServer"
	nextKey             contextKeyType = "Next"
	logKey              contextKeyType = "Log"
)

// WithClientConnection -
//   Wraps 'parent' in a new Context that has the ClientConnection
//   provided in:
//       connection *networkservice.Connection
//   using Context.Value(...) and returns the result.
//   Note: any previously existing ClientConnection will be overwritten.
//
//   Recommended use: in any Request or Close call that creates a ClientConnection, call:
//      ctx = WithClientConnection(ctx)
//   to ensure that the ctx has a ClientConnection
//   In any Request or Close call that consumes a ClientConnection, call:
//	    connectionMap := ClientConnection(ctx)
//   to retrieve the ClientConnection from the context.Context
func WithClientConnection(parent context.Context, connection *networkservice.Connection) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, clientConnectionKey, connection)
}

// ClientConnection -
//    Returns a ClientConnection from:
//      ctx context.Context
//    If any is present, otherwise nil
func ClientConnection(ctx context.Context) *networkservice.Connection {
	conn := ctx.Value(clientConnectionKey)
	if conn == nil {
		return nil
	}
	return conn.(*networkservice.Connection)
}

// withNext -
//    Wraps 'parent' in a new Context that has the Next networkservice.NetworkServiceServer to be called in the chain
//    Should only be set in CompositeEndpoint.Request/Close
func withNext(parent context.Context, next networkservice.NetworkServiceServer) context.Context {
	if parent == nil {
		parent = context.TODO()
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

// withLog -
//   Provides a FieldLogger in context
func withLog(parent context.Context, log logrus.FieldLogger) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, logKey, log)
}

// Log - return FieldLogger from context
func Log(ctx context.Context) logrus.FieldLogger {
	if rv, ok := ctx.Value(logKey).(logrus.FieldLogger); ok {
		return rv
	}
	return logrus.New()
}

// WithMonitorServer -
//   Wraps 'parent' in a new Context that has the local connection Monitor
//   using Context.Value(...) and returns the result.
//   Note: any previously existing MonitorServer will be overwritten.
//
func WithMonitorServer(parent context.Context, monitorServer connectionMonitor.MonitorServer) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, monitorServerKey, monitorServer)
}

// MonitorServer -
//    Returns a MonitorServer from:
//      ctx context.Context
//    If any is present, otherwise nil
func MonitorServer(ctx context.Context) connectionMonitor.MonitorServer {
	value := ctx.Value(monitorServerKey)
	if value == nil {
		return nil
	}
	return value.(connectionMonitor.MonitorServer)
}
