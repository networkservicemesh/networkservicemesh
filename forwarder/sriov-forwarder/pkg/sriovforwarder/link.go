package sriovforwarder

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// LinkStatus defines admin state of the network interface
type LinkStatus uint

const (
	// DOWN is link admin state down
	DOWN LinkStatus = iota
	// UP is link admin state down
	UP
)

// Link represents network interface and specifies operations
// that can be performed on that interface.
type Link interface {
	AddAddress(ip string) error
	DeleteAddress(ip string) error
	MoveToNetns(target netns.NsHandle) error
	SetAdminState(state LinkStatus) error
	SetName(name string) error
}

// vfLink is Link interface implementation for SR-IOV VF interfaces
type vfLink struct {
	link  netlink.Link
	netns netns.NsHandle
}

// GetLink returns a new instance of Link, SRIOV VF representor, based on the PCI
// address and target interface name.
func GetLink(pciAddress, name string, namespaces ...netns.NsHandle) (Link, error) {
	// TODO: add support for shared VF interfaces (like Mellanox NICs)

	attempts := []func(netns.NsHandle, string, string) (netlink.Link, error){
		searchByPCIAddress,
		searchByName,
	}

	// search for link with a matching name or PCI address in the provided namespaces
	for _, ns := range namespaces {
		for _, search := range attempts {
			link, err := search(ns, name, pciAddress)
			if err != nil {
				continue
			}

			if link != nil {
				return &vfLink{
					link:  link,
					netns: ns,
				}, nil
			}
		}
	}

	return nil, errors.Errorf("failed to obtain netlink link matching criteria: name=%s or pciAddress=%s", name, pciAddress)
}

func (vf *vfLink) MoveToNetns(target netns.NsHandle) error {
	// don't do anything if already there
	if vf.netns.Equal(target) {
		return nil
	}

	// set link down
	err := vf.SetAdminState(DOWN)
	if err != nil {
		return errors.Errorf("failed to move link %s to netns: %q", vf.link, err)
	}

	// set netns
	err = netlink.LinkSetNsFd(vf.link, int(target))
	if err != nil {
		return errors.Errorf("failed to move link %s to netns: %q", vf.link, err)
	}

	vf.netns = target

	return nil
}

func (vf *vfLink) AddAddress(ip string) error {
	// parse IP address
	addr, err := netlink.ParseAddr(ip)
	if err != nil {
		return errors.Errorf("failed to parse IP address %q: %s", ip, err)
	}

	// check if address is already assigned
	current, err := netlink.AddrList(vf.link, netlink.FAMILY_ALL)
	if err != nil {
		return errors.Errorf("failed to get current IP address list %q: %s", ip, err)
	}

	for _, existing := range current {
		if addr.Equal(existing) {
			// nothing to do
			return nil
		}
	}

	// add address
	err = netlink.AddrAdd(vf.link, addr)
	if err != nil {
		return errors.Errorf("failed to add IP address %q: %s", ip, err)
	}

	return nil
}

func (vf *vfLink) DeleteAddress(ip string) error {
	// parse IP address
	addr, err := netlink.ParseAddr(ip)
	if err != nil {
		return errors.Errorf("failed to parse IP address %q: %s", ip, err)
	}

	// delete address
	err = netlink.AddrDel(vf.link, addr)
	if err != nil {
		return errors.Errorf("failed to delete IP address %q: %s", ip, err)
	}

	return nil
}

func (vf *vfLink) SetAdminState(state LinkStatus) error {
	switch state {
	case DOWN:
		err := netlink.LinkSetDown(vf.link)
		if err != nil {
			return errors.Errorf("failed to set %s down: %s", vf.link, err)
		}
	case UP:
		err := netlink.LinkSetUp(vf.link)
		if err != nil {
			return errors.Errorf("failed to bring %s up: %s", vf.link, err)
		}
	}

	return nil
}

func (vf *vfLink) SetName(name string) error {
	if vf.link.Attrs().Name != name {
		err := netlink.LinkSetName(vf.link, name)
		if err != nil {
			return errors.Errorf("failed to set interface name to %s: %s", name, err)
		}
	}

	return nil
}

func searchByPCIAddress(ns netns.NsHandle, name, pciAddress string) (netlink.Link, error) {
	// execute in context of the pod's namespace
	err := netns.Set(ns)
	if err != nil {
		return nil, errors.Errorf("failed to enter namespace: %s", err)
	}

	netDir := filepath.Join("/sys/bus/pci/devices", pciAddress, "net")
	if _, err := os.Lstat(netDir); err != nil {
		return nil, errors.Errorf("no net directory under pci device %s: %q", pciAddress, err)
	}

	fInfos, err := ioutil.ReadDir(netDir)
	if err != nil {
		return nil, errors.Errorf("failed to read net directory %s: %q", netDir, err)
	}

	names := make([]string, 0)
	for _, f := range fInfos {
		names = append(names, f.Name())
	}

	if len(names) == 0 {
		return nil, errors.Errorf("no links with PCI address %s found", pciAddress)
	}

	link, err := netlink.LinkByName(names[0])
	if err != nil {
		return nil, errors.Errorf("error getting VF netdevice with PCI address %s", pciAddress)
	}

	return link, nil
}

func searchByName(ns netns.NsHandle, name, pciAddress string) (netlink.Link, error) {
	// execute in context of the pod's namespace
	err := netns.Set(ns)
	if err != nil {
		return nil, errors.Errorf("failed to switch to namespace: %s", err)
	}

	// get link
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, errors.Errorf("failed to get VF link with name %s", name)
	}

	return link, nil
}
