package local

import (
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

const (
	/* VETH pairs are used only for local connections(same node), so we can use a larger MTU size as there's no multi-node connection */
	cVETHMTU = 16000
)

// Connect -
type Connect struct{}

// NewConnect -
func NewConnect() *Connect {
	return &Connect{}
}

// CreateInterfaces - creates local interfaces pair
func (c *Connect) CreateInterfaces(srcName, dstName string) error {
	/* Create the VETH pair - host namespace */
	if err := netlink.LinkAdd(newVETH(srcName, dstName)); err != nil {
		return errors.Errorf("failed to create VETH pair - %v", err)
	}
	return nil
}

// CreateInterface - deletes interface to remote connection
func (c *Connect) DeleteInterfaces(ifaceName string) error {
	/* Get a link object for the interface */
	ifaceLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return errors.Errorf("failed to get link for %q - %v", ifaceName, err)
	}

	/* Delete the VETH pair - host namespace */
	if err := netlink.LinkDel(ifaceLink); err != nil {
		return errors.Errorf("local: failed to delete the VETH pair - %v", err)
	}

	return nil
}

func newVETH(srcName, dstName string) *netlink.Veth {
	/* Populate the VETH interface configuration */
	return &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: srcName,
			MTU:  cVETHMTU,
		},
		PeerName: dstName,
	}
}
