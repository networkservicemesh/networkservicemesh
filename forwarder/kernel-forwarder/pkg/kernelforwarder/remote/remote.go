package remote

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
)

const (
	INCOMING = iota
	OUTGOING = iota
)

// CreateRemoteInterface - creates interface to remote connection
func CreateRemoteInterface(ifaceName string, remoteConnection *connection.Connection, direction uint8) error {
	return createVXLANInterface(ifaceName, remoteConnection, direction)
}

// CreateRemoteInterface - deletes interface to remote connection
func DeleteRemoteInterface(ifaceName string) error {
	return deleteVXLANInterface(ifaceName)
}