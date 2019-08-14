package plugins

import (
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
)

// NewConnectionInfo creates a ConnectionInfo instance
func NewConnectionInfo(conn connection.Connection) *ConnectionInfo {
	c := &ConnectionInfo{}
	c.SetConnection(conn)
	return c
}

// GetConnection returns connection
func (c *ConnectionInfo) GetConnection() connection.Connection {
	if c.GetLocalConnection() != nil {
		return c.GetLocalConnection()
	}
	return c.GetRemoteConnection()
}

// SetConnection sets connection
func (c *ConnectionInfo) SetConnection(conn connection.Connection) {
	if conn.IsRemote() {
		c.Conn = &ConnectionInfo_RemoteConnection{
			RemoteConnection: conn.(*remote.Connection),
		}
	} else {
		c.Conn = &ConnectionInfo_LocalConnection{
			LocalConnection: conn.(*local.Connection),
		}
	}
}
