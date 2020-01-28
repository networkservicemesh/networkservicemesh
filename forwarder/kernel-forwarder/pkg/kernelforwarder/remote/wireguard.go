package remote

import (
	"net"
	"strconv"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"

)

// CreateVXLANInterface creates a VXLAN interface
func createWireguardInterface(ifaceName string, remoteConnection *connection.Connection, direction uint8) error {
	/* Create interface - host namespace */
	srcIP := net.ParseIP(remoteConnection.GetMechanism().GetParameters()[vxlan.SrcIP])
	dstIP := net.ParseIP(remoteConnection.GetMechanism().GetParameters()[vxlan.DstIP])
	vni, _ := strconv.Atoi(remoteConnection.GetMechanism().GetParameters()[vxlan.DstIP])

	var localIP net.IP
	var remoteIP net.IP
	if direction == INCOMING {
		localIP = dstIP
		remoteIP = srcIP
	} else {
		localIP = srcIP
		remoteIP = dstIP
	}

	if err := netlink.LinkAdd(newVXLAN(ifaceName, localIP, remoteIP, vni)); err != nil {
		return errors.Wrapf(err, "failed to create VXLAN interface")
	}
	return nil
}

func deleteWireguardInterface(ifaceName string) error {
	/* Get a link object for interface */
	ifaceLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return errors.Errorf("failed to get link for %q - %v", ifaceName, err)
	}

	/* Delete the VXLAN interface - host namespace */
	if err = netlink.LinkDel(ifaceLink); err != nil {
		err = errors.Errorf("failed to delete VXLAN interface - %v", err)
	}

	return nil
}
