package connect

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
)

const (
	clientConnectionKey contextKeyType = "ClientConnection"
)

type contextKeyType string

// withClientConnection -
//    Wraps 'parent' in a new Context that has the ClientConnection
func withClientConnection(parent context.Context, conn *connection.Connection) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, clientConnectionKey, conn)
}

// ClientUrl -
//   Returns the ClientUrl
func ClientConnection(ctx context.Context) *connection.Connection {
	if rv, ok := ctx.Value(clientConnectionKey).(*connection.Connection); ok {
		return rv
	}
	return nil
}
