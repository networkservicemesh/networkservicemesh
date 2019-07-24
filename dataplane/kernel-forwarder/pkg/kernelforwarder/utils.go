// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kernelforwarder

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"

	"fmt"
	"net"
	"runtime"
	"strconv"

	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// Kernel forwarding plane related constants
const (
	cLOCAL    = iota
	cINCOMING = iota
	cOUTGOING = iota
)

const (
	cCONNECT    = true
	cDISCONNECT = false
	/* VETH pairs are used only for local connections(same node), so we can use a larger MTU size as there's no multi-node connection */
	cVETHMTU = 16000
)

type connectionConfig struct {
	srcNsPath  string
	dstNsPath  string
	srcName    string
	dstName    string
	srcIP      string
	dstIP      string
	srcIPVXLAN net.IP
	dstIPVXLAN net.IP
	vni        int
}

func handleLocalConnection(crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	logrus.Info("Incoming connection - local source/local destination")
	/* 1. Get the connection configuration */
	cfg, err := newConnectionConfig(crossConnect, cLOCAL)
	if err != nil {
		logrus.Errorf("Failed to get the configuration for local connection - %v", err)
		return crossConnect, err
	}
	if connect {
		/* 2. Create a connection */
		err = createLocalConnection(cfg)
		if err != nil {
			logrus.Errorf("Failed to create local connection - %v", err)
		}
	} else {
		/* 3. Delete a connection */
		err = deleteLocalConnection(cfg)
		if err != nil {
			logrus.Errorf("Failed to delete local connection - %v", err)
		}
	}
	return crossConnect, nil
}

func createLocalConnection(cfg *connectionConfig) error {
	logrus.Info("Creating local connection...")
	/* 1. Get handlers for source and destination namespaces */
	srcNsHandle, err := netns.GetFromPath(cfg.srcNsPath)
	defer srcNsHandle.Close()
	if err != nil {
		logrus.Errorf("failed to get source namespace handler from path - %v", err)
		return err
	}
	dstNsHandle, err := netns.GetFromPath(cfg.dstNsPath)
	defer dstNsHandle.Close()
	if err != nil {
		logrus.Errorf("failed to get destination namespace handler from path - %v", err)
		return err
	}

	/* 2. Prepare interface - VETH */
	iface := newVETH(cfg.srcName, cfg.dstName)

	/* 3. Create the VETH pair - host namespace */
	if err = netlink.LinkAdd(iface); err != nil {
		logrus.Errorf("failed to create the VETH pair - %v", err)
		return err
	}

	/* 4. Setup interface - source namespace */
	if err = setupLinkInNs(srcNsHandle, cfg.srcName, cfg.srcIP, true); err != nil {
		logrus.Errorf("failed to setup container interface %q: %v", cfg.srcName, err)
		return err
	}

	/* 5. Setup interface - destination namespace */
	if err = setupLinkInNs(dstNsHandle, cfg.dstName, cfg.dstIP, true); err != nil {
		logrus.Errorf("failed to setup container interface %q: %v", cfg.dstName, err)
		return err
	}
	return nil
}

func deleteLocalConnection(cfg *connectionConfig) error {
	logrus.Info("Delete local connection...")
	/* 1. Get handlers for source and destination namespaces */
	srcNsHandle, err := netns.GetFromPath(cfg.srcNsPath)
	defer srcNsHandle.Close()
	if err != nil {
		logrus.Errorf("failed to get source namespace handler from path - %v", err)
		return err
	}
	dstNsHandle, err := netns.GetFromPath(cfg.dstNsPath)
	defer dstNsHandle.Close()
	if err != nil {
		logrus.Errorf("failed to get destination namespace handler from path - %v", err)
		return err
	}

	/* 2. Extract the interface - source namespace */
	if err = setupLinkInNs(srcNsHandle, cfg.srcName, cfg.srcIP, false); err != nil {
		logrus.Errorf("failed to setup container interface %q: %v", cfg.srcName, err)
		return err
	}

	/* 3. Extract the interface - destination namespace */
	if err = setupLinkInNs(dstNsHandle, cfg.dstName, cfg.dstIP, false); err != nil {
		logrus.Errorf("failed to setup container interface %q: %v", cfg.dstName, err)
		return err
	}

	/* 4. Get a link object for the interface */
	ifaceLink, err := netlink.LinkByName(cfg.srcName)
	if err != nil {
		logrus.Errorf("failed to get link for %q - %v", cfg.srcName, err)
		return err
	}

	/* 5. Delete the VETH pair - host namespace */
	if err := netlink.LinkDel(ifaceLink); err != nil {
		logrus.Errorf("failed to delete the VETH pair - %v", err)
		return err
	}
	return nil
}

func handleRemoteConnection(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	if crossConnect.GetRemoteSource().GetMechanism().GetType() == remote.MechanismType_VXLAN &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE {
		/* 1. Incoming remote connection */
		return handleIncoming(egress, crossConnect, connect)
	} else if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE &&
		crossConnect.GetRemoteDestination().GetMechanism().GetType() == remote.MechanismType_VXLAN {
		/* 2. Outgoing remote connection */
		return handleOutgoing(egress, crossConnect, connect)
	}
	logrus.Errorf("invalid remote connection type")
	return crossConnect, fmt.Errorf("invalid remote connection type")
}

func handleIncoming(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	logrus.Info("Incoming connection - remote source/local destination")
	/* 1. Get the connection configuration */
	cfg, err := newConnectionConfig(crossConnect, cINCOMING)
	if err != nil {
		logrus.Errorf("failed to get the configuration for remote connection - %v", err)
		return crossConnect, err
	}
	if connect {
		/* 2. Create a connection */
		err = createRemoteConnection(cfg.dstNsPath, cfg.dstName, cfg.dstIP, egress.Name(), egress.SrcIPNet().IP, cfg.srcIPVXLAN, cfg.vni)
		if err != nil {
			logrus.Errorf("failed to create remote connection - %v", err)
		}
	} else {
		/* 3. Delete a connection */
		err = deleteRemoteConnection(cfg.dstNsPath, cfg.dstName)
		if err != nil {
			logrus.Errorf("failed to delete remote connection - %v", err)
		}
	}
	return crossConnect, err
}

func handleOutgoing(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	logrus.Info("Outgoing connection - local source/remote destination")
	/* 1. Get the connection configuration */
	cfg, err := newConnectionConfig(crossConnect, cOUTGOING)
	if err != nil {
		logrus.Errorf("failed to get the configuration for remote connection - %v", err)
		return crossConnect, err
	}
	if connect {
		/* 2. Create a connection */
		err = createRemoteConnection(cfg.srcNsPath, cfg.srcName, cfg.srcIP, egress.Name(), egress.SrcIPNet().IP, cfg.dstIPVXLAN, cfg.vni)
		if err != nil {
			logrus.Errorf("failed to create remote connection - %v", err)
		}
	} else {
		/* 3. Delete a connection */
		err = deleteRemoteConnection(cfg.srcNsPath, cfg.srcName)
		if err != nil {
			logrus.Errorf("failed to delete remote connection - %v", err)
		}
	}
	return crossConnect, err
}

func createRemoteConnection(nsPath, ifaceName, ifaceIP, egressName string, egressIP, remoteIP net.IP, vni int) error {
	logrus.Info("Creating remote connection...")
	/* 1. Get handler for container namespace */
	containerNs, err := netns.GetFromPath(nsPath)
	defer containerNs.Close()
	if err != nil {
		logrus.Errorf("failed to get namespace handler from path - %v", err)
		return err
	}

	/* 2. Prepare interface - VXLAN */
	iface, err := newVXLAN(ifaceName, egressName, egressIP, remoteIP, vni)
	if err != nil {
		logrus.Errorf("failed to get VXLAN interface configuration - %v", err)
		return err
	}

	/* 3. Create interface - host namespace */
	if err = netlink.LinkAdd(iface); err != nil {
		logrus.Errorf("failed to create VXLAN interface - %v", err)
	}

	/* 4. Setup interface */
	if err = setupLinkInNs(containerNs, ifaceName, ifaceIP, true); err != nil {
		logrus.Errorf("failed to setup container interface %q: %v", ifaceName, err)
		return err
	}
	return nil
}

func setupLinkInNs(containerNs netns.NsHandle, ifaceName, ifaceIP string, inject bool) error {
	if inject {
		/* 1. Get a link object for the interface */
		ifaceLink, err := netlink.LinkByName(ifaceName)
		if err != nil {
			logrus.Errorf("failed to get link for %q - %v", ifaceName, err)
			return err
		}
		/* 2. Inject the interface into the desired container namespace */
		if err = netlink.LinkSetNsFd(ifaceLink, int(containerNs)); err != nil {
			logrus.Errorf("failed to inject %q in namespace - %v", ifaceName, err)
			return err
		}
	}
	/* 3. Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	/* 4. Save the current network namespace */
	currentNs, err := netns.Get()
	defer currentNs.Close()
	if err != nil {
		logrus.Errorf("failed to get current namespace: %v", err)
		return err
	}
	/* 5. Switch to the new namespace */
	if err = netns.Set(containerNs); err != nil {
		logrus.Errorf("failed to switch to container namespace: %v", err)
		return err
	}
	defer containerNs.Close()
	/* 6. Get a link for the interface name */
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		logrus.Errorf("failed to lookup %q, %v", ifaceName, err)
		return err
	}
	if inject {
		var addr *netlink.Addr
		/* 7. Parse the IP address */
		addr, err = netlink.ParseAddr(ifaceIP)
		if err != nil {
			logrus.Errorf("failed to parse IP %q: %v", ifaceIP, err)
			return err
		}
		/* 8. Set IP address */
		if err = netlink.AddrAdd(link, addr); err != nil {
			logrus.Errorf("failed to set IP %q: %v", ifaceIP, err)
			return err
		}
		/* 9. Bring the interface UP */
		if err = netlink.LinkSetUp(link); err != nil {
			logrus.Errorf("failed to bring %q up: %v", ifaceName, err)
			return err
		}
	} else {
		/* 9. Bring the interface DOWN */
		if err = netlink.LinkSetDown(link); err != nil {
			logrus.Errorf("failed to bring %q down: %v", ifaceName, err)
			return err
		}
		/* 2. Inject the interface back into the host namespace */
		if err = netlink.LinkSetNsFd(link, int(currentNs)); err != nil {
			logrus.Errorf("failed to inject %q bach to host namespace - %v", ifaceName, err)
			return err
		}
	}
	/* 10. Switch back to the original namespace */
	if err = netns.Set(currentNs); err != nil {
		logrus.Errorf("failed to switch back to original namespace: %v", err)
		return err
	}
	return nil
}

func deleteRemoteConnection(nsPath, ifaceName string) error {
	logrus.Info("Deleting remote connection...")
	/* 1. Get handler for container namespace */
	containerNs, err := netns.GetFromPath(nsPath)
	defer containerNs.Close()
	if err != nil {
		logrus.Errorf("failed to get namespace handler from path - %v", err)
		return err
	}

	/* 2. Setup interface */
	if err = setupLinkInNs(containerNs, ifaceName, "", false); err != nil {
		logrus.Errorf("failed to setup container interface %q: %v", ifaceName, err)
		return err
	}

	/* 3. Get a link object for the interface */
	ifaceLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		logrus.Errorf("failed to get link for %q - %v", ifaceName, err)
		return err
	}

	/* 4. Delete the VXLAN interface - host namespace */
	if err := netlink.LinkDel(ifaceLink); err != nil {
		logrus.Errorf("failed to delete the VXLAN - %v", err)
		return err
	}
	return nil
}

func newConnectionConfig(crossConnect *crossconnect.CrossConnect, connType uint8) (*connectionConfig, error) {
	switch connType {
	case cLOCAL:
		srcNsPath, err := crossConnect.GetLocalSource().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("failed to get source namespace path - %v", err)
			return nil, err
		}
		dstNsPath, err := crossConnect.GetLocalDestination().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("failed to get destination namespace path - %v", err)
			return nil, err
		}
		return &connectionConfig{
			srcNsPath: srcNsPath,
			dstNsPath: dstNsPath,
			srcName:   crossConnect.GetLocalSource().GetMechanism().GetParameters()[local.InterfaceNameKey],
			dstName:   crossConnect.GetLocalDestination().GetMechanism().GetParameters()[local.InterfaceNameKey],
			srcIP:     crossConnect.GetLocalSource().GetContext().GetIpContext().GetSrcIpAddr(),
			dstIP:     crossConnect.GetLocalSource().GetContext().GetIpContext().GetDstIpAddr(),
		}, nil
	case cINCOMING:
		dstNsPath, err := crossConnect.GetLocalDestination().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("failed to get destination namespace path - %v", err)
			return nil, err
		}
		vni, _ := strconv.Atoi(crossConnect.GetRemoteSource().GetMechanism().GetParameters()[remote.VXLANVNI])
		return &connectionConfig{
			dstNsPath:  dstNsPath,
			dstName:    crossConnect.GetLocalDestination().GetMechanism().GetParameters()[local.InterfaceNameKey],
			dstIP:      crossConnect.GetLocalDestination().GetContext().GetIpContext().GetDstIpAddr(),
			srcIPVXLAN: net.ParseIP(crossConnect.GetRemoteSource().GetMechanism().GetParameters()[remote.VXLANSrcIP]),
			dstIPVXLAN: net.ParseIP(crossConnect.GetRemoteSource().GetMechanism().GetParameters()[remote.VXLANDstIP]),
			vni:        vni,
		}, nil
	case cOUTGOING:
		srcNsPath, err := crossConnect.GetLocalSource().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("failed to get destination namespace path - %v", err)
			return nil, err
		}
		vni, _ := strconv.Atoi(crossConnect.GetRemoteDestination().GetMechanism().GetParameters()[remote.VXLANVNI])
		return &connectionConfig{
			srcNsPath:  srcNsPath,
			srcName:    crossConnect.GetLocalSource().GetMechanism().GetParameters()[local.InterfaceNameKey],
			srcIP:      crossConnect.GetLocalSource().GetContext().GetIpContext().GetSrcIpAddr(),
			srcIPVXLAN: net.ParseIP(crossConnect.GetRemoteDestination().GetMechanism().GetParameters()[remote.VXLANSrcIP]),
			dstIPVXLAN: net.ParseIP(crossConnect.GetRemoteDestination().GetMechanism().GetParameters()[remote.VXLANDstIP]),
			vni:        vni,
		}, nil
	default:
		logrus.Error("connection configuration: invalid connection type")
		return nil, fmt.Errorf("invalid connection type")
	}
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

func newVXLAN(ifaceName, egressName string, egressIP, remoteIP net.IP, vni int) (*netlink.Vxlan, error) {
	/* Get a link object for the egress interface on the host */
	egressLink, err := netlink.LinkByName(egressName)
	if err != nil {
		logrus.Errorf("Failed to get egress VXLAN interface - %v", err)
		return nil, err
	}
	/* Populate the VXLAN interface configuration */
	return &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: ifaceName,
		},
		VxlanId:      vni,
		VtepDevIndex: egressLink.Attrs().Index,
		Group:        remoteIP,
		SrcAddr:      egressIP,
	}, nil
}
