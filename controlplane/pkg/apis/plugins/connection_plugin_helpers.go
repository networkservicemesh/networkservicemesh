package plugins

import (
	"github.com/golang/protobuf/proto"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
)

// NewConnectionWrapper creates a ConnectionWrapper instance
func NewConnectionWrapper(conn connection.Connection) *ConnectionWrapper {
	w := &ConnectionWrapper{}
	w.SetConnection(conn)
	return w
}

// GetConnection returns connection
func (w *ConnectionWrapper) GetConnection() connection.Connection {
	if w.GetLocalConnection() != nil {
		return w.GetLocalConnection()
	}
	return w.GetRemoteConnection()
}

// SetConnection sets connection
func (w *ConnectionWrapper) SetConnection(conn connection.Connection) {
	if conn.IsRemote() {
		w.Conn = &ConnectionWrapper_RemoteConnection{
			RemoteConnection: conn.(*remote.Connection),
		}
	} else {
		w.Conn = &ConnectionWrapper_LocalConnection{
			LocalConnection: conn.(*local.Connection),
		}
	}
}

// Clone clones wrapper with connection
func (w *ConnectionWrapper) Clone() *ConnectionWrapper {
	return proto.Clone(w).(*ConnectionWrapper)
}
