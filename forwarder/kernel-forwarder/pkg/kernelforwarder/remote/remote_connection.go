package remote

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
)

const (
	INCOMING = iota
	OUTGOING = iota
)

// CreateRemoteInterface - creating interface to remote connection
func CreateRemoteInterface(ifaceName string, remoteConnection *connection.Connection, direction uint8) error {
	return createVXLANInterface(ifaceName, remoteConnection, direction)
}