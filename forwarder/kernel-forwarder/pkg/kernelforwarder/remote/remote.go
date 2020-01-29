// Package remote - controlling remote mechanisms interfaces
package remote

import (
	"github.com/pkg/errors"
	wg "golang.zx2c4.com/wireguard/device"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/wireguard"
)

// INCOMING, OUTGOING - packet direction constants
const (
	INCOMING = iota
	OUTGOING = iota
)

// Connect - struct with remote mechanism interfaces creation and deletion methods
type Connect struct {
	wireguardDevices map[string]wg.Device
}

// NewConnect - creates instance of remote Connect
func NewConnect() *Connect {
	return &Connect{}
}

// CreateInterface - creates interface to remote connection
func (c *Connect) CreateInterface(ifaceName string, remoteConnection *connection.Connection, direction uint8) error {
	switch remoteConnection.GetMechanism().GetType() {
	case vxlan.MECHANISM:
		return c.createVXLANInterface(ifaceName, remoteConnection, direction)
	case wireguard.MECHANISM:
		return c.createWireguardInterface(ifaceName, remoteConnection, direction)
	}
	return errors.Errorf("unknown remote mechanism - %v", remoteConnection.GetMechanism().GetType())
}

// DeleteInterface - deletes interface to remote connection
func (c *Connect) DeleteInterface(ifaceName string, remoteConnection *connection.Connection) error {
	switch remoteConnection.GetMechanism().GetType() {
	case vxlan.MECHANISM:
		return c.deleteVXLANInterface(ifaceName)
	case wireguard.MECHANISM:
		return c.deleteWireguardInterface(ifaceName)
	}
	return errors.Errorf("unknown remote mechanism - %v", remoteConnection.GetMechanism().GetType())
}
