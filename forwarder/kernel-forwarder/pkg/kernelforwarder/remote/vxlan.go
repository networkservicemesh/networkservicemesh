package remote

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"net"
	"strconv"
)

// CreateVXLANInterface creates a VXLAN interface
func createVXLANInterface(ifaceName string, remoteConnection *connection.Connection, direction uint8) error {
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

// newVXLAN returns a VXLAN interface instance
func newVXLAN(ifaceName string, egressIP, remoteIP net.IP, vni int) *netlink.Vxlan {
	/* Populate the VXLAN interface configuration */
	return &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: ifaceName,
		},
		VxlanId: vni,
		Group:   remoteIP,
		SrcAddr: egressIP,
	}
}
