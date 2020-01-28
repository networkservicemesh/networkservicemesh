package remote

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/wireguard"
	"github.com/pkg/errors"
)

const (
	INCOMING = iota
	OUTGOING = iota
)

// SetupRemoteInterface - creates interface to remote connection
func SetupRemoteInterface(ifaceName string, remoteConnection *connection.Connection, direction uint8) error {
	switch remoteConnection.GetMechanism().GetType() {
	case vxlan.MECHANISM:
		return createVXLANInterface(ifaceName, remoteConnection, direction)
	case wireguard.MECHANISM:
		return createWireguardInterface(ifaceName, remoteConnection, direction)
	}
	return errors.Errorf("unknown remote mechanism - %v", remoteConnection.GetMechanism().GetType())
}

// SetupRemoteInterface - deletes interface to remote connection
func DeleteRemoteInterface(ifaceName string, remoteConnection *connection.Connection) error {
	switch remoteConnection.GetMechanism().GetType() {
	case vxlan.MECHANISM:
		return deleteVXLANInterface(ifaceName)
	case wireguard.MECHANISM:
		return deleteWireguardInterface(ifaceName)
	}
	return errors.Errorf("unknown remote mechanism - %v", remoteConnection.GetMechanism().GetType())
}