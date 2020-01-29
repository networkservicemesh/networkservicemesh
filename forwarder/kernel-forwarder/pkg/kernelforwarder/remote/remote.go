package remote

import (
	"github.com/pkg/errors"
	wg "golang.zx2c4.com/wireguard/device"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/wireguard"
)

const (
	INCOMING = iota
	OUTGOING = iota
)

// Connect -
type Connect struct {
	wireguardDevices map[string]wg.Device
}

// NewConnect -
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

// CreateInterface - deletes interface to remote connection
func (c *Connect) DeleteInterface(ifaceName string, remoteConnection *connection.Connection) error {
	switch remoteConnection.GetMechanism().GetType() {
	case vxlan.MECHANISM:
		return c.deleteVXLANInterface(ifaceName)
	case wireguard.MECHANISM:
		return c.deleteWireguardInterface(ifaceName)
	}
	return errors.Errorf("unknown remote mechanism - %v", remoteConnection.GetMechanism().GetType())
}
