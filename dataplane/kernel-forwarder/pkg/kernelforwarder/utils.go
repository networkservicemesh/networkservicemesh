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

	"github.com/davecgh/go-spew/spew"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// Kernel forwarding plane related constants
const (
	cLOCAL      = 1
	cINCOMING   = 2
	cOUTGOING   = 3
	cCONNECT    = true
	cDISCONNECT = false
)

type connectionConfig struct {
	srcNsPath  string
	dstNsPath  string
	srcName    string
	dstName    string
	srcIP      string
	dstIP      string
	srcIPVXLAN string
	dstIPVXLAN string
	vni        int
}

func handleLocalConnection(crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	/* 1. Get the connection configuration */
	cfg, err := getConnectionConfig(crossConnect, cLOCAL)
	if err != nil {
		logrus.Errorf("Failed to get the configuration for local connection - %v", err)
		return crossConnect, err
	}
	/* 2. Create a connection */
	if connect {
		err = createLocalConnection(cfg)
		if err != nil {
			logrus.Errorf("Failed to create local connection - %v", err)
			return crossConnect, err
		}
	}
	/* 3. Delete a connection */
	err = deleteLocalConnection(cfg)
	if err != nil {
		logrus.Errorf("Failed to delete local connection - %v", err)
		return crossConnect, err
	}
	return crossConnect, nil
}

func createLocalConnection(cfg *connectionConfig) error {
	/* 1. Get namespace handlers from their path - source and destination */
	srcNsHandle, err := netns.GetFromPath(cfg.srcNsPath)
	defer srcNsHandle.Close()
	if err != nil {
		logrus.Errorf("Failed to get source namespace handler from path - %v", err)
		return err
	}
	dstNsHandle, err := netns.GetFromPath(cfg.dstNsPath)
	defer dstNsHandle.Close()
	if err != nil {
		logrus.Errorf("Failed to get destination namespace handler from path - %v", err)
		return err
	}
	/* 2. Create a VETH pair and inject each end in the corresponding namespace */
	if err = createVETH(cfg, srcNsHandle, dstNsHandle); err != nil {
		logrus.Errorf("Failed to create the VETH pair - %v", err)
		return err
	}
	/* 3. Bring up and configure each pair end with its IP address */
	setupVETHEnd(srcNsHandle, cfg.srcName, cfg.srcIP)
	setupVETHEnd(dstNsHandle, cfg.dstName, cfg.dstIP)
	return nil
}

func deleteLocalConnection(cfg *connectionConfig) error {
	logrus.Errorf("Delete for local connection is not supported yet")
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
	} else {
		logrus.Errorf("Invalid remote connection type")
	}
	return crossConnect, nil
}

func handleIncoming(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	logrus.Info("Incoming connection - remote source/local destination")
	/* 1. Get the connection configuration */
	cfg, err := getConnectionConfig(crossConnect, cINCOMING)
	logrus.Info(spew.Sdump(cfg), egress.Name())
	if err != nil {
		logrus.Errorf("Failed to get the configuration for remote connection - %v", err)
		return crossConnect, err
	}
	/* 2. Create a connection */
	if connect {
		err = createRemoteConnection(cfg.dstNsPath, cfg.dstName, cfg.dstIP, egress.Name(), egress.SrcIPNet().IP, cfg.vni)
		if err != nil {
			logrus.Errorf("Failed to create remote connection - %v", err)
			return crossConnect, err
		}
	}
	/* 3. Delete a connection */
	err = deleteRemoteConnection(cfg.dstNsPath, cfg.dstName, cfg.dstIP, egress.Name(), egress.SrcIPNet().IP, cfg.vni)
	if err != nil {
		logrus.Errorf("Failed to delete remote connection - %v", err)
		return crossConnect, err
	}
	return crossConnect, nil
}

func handleOutgoing(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	logrus.Info("Outgoing connection - local source/remote destination")
	/* 1. Get the connection configuration */
	cfg, err := getConnectionConfig(crossConnect, cOUTGOING)
	logrus.Info(spew.Sdump(cfg), egress.Name())
	if err != nil {
		logrus.Errorf("Failed to get the configuration for remote connection - %v", err)
		return crossConnect, err
	}
	/* 2. Create a connection */
	if connect {
		err = createRemoteConnection(cfg.srcNsPath, cfg.srcName, cfg.srcIP, egress.Name(), egress.SrcIPNet().IP, cfg.vni)
		if err != nil {
			logrus.Errorf("Failed to create remote connection - %v", err)
			return crossConnect, err
		}
	}
	/* 3. Delete a connection */
	err = deleteRemoteConnection(cfg.srcNsPath, cfg.srcName, cfg.srcIP, egress.Name(), egress.SrcIPNet().IP, cfg.vni)
	if err != nil {
		logrus.Errorf("Failed to delete remote connection - %v", err)
		return crossConnect, err
	}
	return crossConnect, nil
}

func createRemoteConnection(NsPath, ifaceName, ifaceIP, egressName string, egressIP net.IP, vni int) error {
	/* 1. Get namespace handler from path */
	srcNsHandle, err := netns.GetFromPath(NsPath)
	defer srcNsHandle.Close()
	if err != nil {
		logrus.Errorf("Failed to get namespace handler from path - %v", err)
		return err
	}

	/* 2. Get the host link object from the egress interface */
	egressLink, err := netlink.LinkByName(egressName)
	if err != nil {
		logrus.Errorf("Failed to get egress VXLAN interface - %v", err)
	}

	/* 3. Populate the VXLAN interface configuration */
	vxlan := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: ifaceName,
		},
		VxlanId:        vni,
		VtepDevIndex:   egressLink.Attrs().Index,
		SrcAddr:        egressIP,
		Learning:       true,
		L2miss:         true,
		L3miss:         true,
		UDP6ZeroCSumTx: true,
		UDP6ZeroCSumRx: true,
		GBP:            true,
	}
	logrus.Info(spew.Sdump(vxlan))

	/* 4. Create the VXLAN interface */
	if err := netlink.LinkAdd(vxlan); err != nil {
		logrus.Errorf("Failed to create VXLAN interface - %v", err)
	}

	/* 5. Get a link object for it */
	egressLink, err = netlink.LinkByName(ifaceName)
	if err != nil {
		logrus.Errorf("Failed to get link for the created VXLAN interface - %v", err)
	}
	nsHandle := srcNsHandle

	/* 6. Inject the VXLAN interface into the desired namespace */
	if err = netlink.LinkSetNsFd(egressLink, int(nsHandle)); err != nil {
		logrus.Errorf("Failed to inject the VXLAN end in the namespace - %v", err)
	}

	/* 7. Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	/* 8. Save the current network namespace */
	oNsHandle, _ := netns.Get()
	defer oNsHandle.Close()

	/* 9. Switch to the new namespace */
	netns.Set(nsHandle)
	defer nsHandle.Close()

	/* 10. Get a link for the interface name */
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		logrus.Errorf("Failed to lookup %q, %v", ifaceName, err)
	}

	/* 11. Setup the interface with an IP address */
	addr, _ := netlink.ParseAddr(ifaceIP)
	netlink.AddrAdd(link, addr)

	/* 12. Bring the interface UP */
	if err = netlink.LinkSetUp(link); err != nil {
		logrus.Errorf("Failed to set %q up: %v", ifaceName, err)
	}

	/* 13. Switch back to the original namespace */
	netns.Set(oNsHandle)
	return nil
}

func deleteRemoteConnection(NsPath, ifaceName, ifaceIP, egressName string, egressIP net.IP, vni int) error {
	logrus.Errorf("Delete for remote connection is not supported yet")
	return nil
}

func getConnectionConfig(crossConnect *crossconnect.CrossConnect, connType uint8) (*connectionConfig, error) {
	switch connType {
	case cLOCAL:
		srcNsPath, err := crossConnect.GetLocalSource().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("Failed to get source namespace path - %v", err)
			return nil, err
		}
		dstNsPath, err := crossConnect.GetLocalDestination().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("Failed to get destination namespace path - %v", err)
			return nil, err
		}
		return &connectionConfig{
			srcNsPath: srcNsPath,
			dstNsPath: dstNsPath,
			srcName:   crossConnect.GetLocalSource().GetMechanism().GetParameters()[local.InterfaceNameKey],
			dstName:   crossConnect.GetLocalDestination().GetMechanism().GetParameters()[local.InterfaceNameKey],
			srcIP:     crossConnect.GetLocalSource().GetContext().GetSrcIpAddr(),
			dstIP:     crossConnect.GetLocalSource().GetContext().GetDstIpAddr(),
		}, nil
	case cINCOMING:
		dstNsPath, err := crossConnect.GetLocalDestination().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("Failed to get destination namespace path - %v", err)
			return nil, err
		}
		vni, _ := strconv.Atoi(crossConnect.GetRemoteSource().GetMechanism().GetParameters()[remote.VXLANVNI])
		return &connectionConfig{
			dstNsPath:  dstNsPath,
			dstName:    crossConnect.GetLocalDestination().GetMechanism().GetParameters()[local.InterfaceNameKey],
			dstIP:      crossConnect.GetLocalDestination().GetContext().GetDstIpAddr(),
			srcIPVXLAN: crossConnect.GetRemoteSource().GetMechanism().GetParameters()[remote.VXLANSrcIP],
			dstIPVXLAN: crossConnect.GetRemoteSource().GetMechanism().GetParameters()[remote.VXLANDstIP],
			vni:        vni,
		}, nil
	case cOUTGOING:
		srcNsPath, err := crossConnect.GetLocalSource().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("Failed to get destination namespace path - %v", err)
			return nil, err
		}
		vni, _ := strconv.Atoi(crossConnect.GetRemoteDestination().GetMechanism().GetParameters()[remote.VXLANVNI])
		return &connectionConfig{
			srcNsPath:  srcNsPath,
			srcName:    crossConnect.GetLocalSource().GetMechanism().GetParameters()[local.InterfaceNameKey],
			srcIP:      crossConnect.GetLocalSource().GetContext().GetSrcIpAddr(),
			srcIPVXLAN: crossConnect.GetRemoteDestination().GetMechanism().GetParameters()[remote.VXLANSrcIP],
			dstIPVXLAN: crossConnect.GetRemoteDestination().GetMechanism().GetParameters()[remote.VXLANDstIP],
			vni:        vni,
		}, nil
	default:
		logrus.Error("Connection configuration: invalid connection type")
		return nil, fmt.Errorf("Invalid connection type")
	}
}

func setupVETHEnd(nsHandle netns.NsHandle, ifName, addrIP string) {
	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	/* Save the current network namespace */
	oNsHandle, _ := netns.Get()
	defer oNsHandle.Close()

	/* Switch to the new namespace */
	netns.Set(nsHandle)
	defer nsHandle.Close()

	/* Get a link for the interface name */
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		logrus.Errorf("Failed to lookup %q, %v", ifName, err)
	}

	/* Setup the interface with an IP address */
	addr, _ := netlink.ParseAddr(addrIP)
	netlink.AddrAdd(link, addr)

	/* Bring the interface UP */
	if err = netlink.LinkSetUp(link); err != nil {
		logrus.Errorf("Failed to set %q up: %v", ifName, err)
	}

	/* Switch back to the original namespace */
	netns.Set(oNsHandle)
}

func createVETH(cfg *connectionConfig, srcNsHandle, dstNsHandle netns.NsHandle) error {
	/* Initial VETH configuration */
	cfgVETH := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: cfg.srcName,
			MTU:  16000,
		},
		PeerName: cfg.dstName,
	}

	/* Create the VETH pair - host namespace */
	if err := netlink.LinkAdd(cfgVETH); err != nil {
		logrus.Errorf("Failed to create the VETH pair - %v", err)
		return err
	}

	/* Get a link for each VETH pair ends */
	srcLink, err := netlink.LinkByName(cfg.srcName)
	if err != nil {
		logrus.Errorf("Failed to get source link from name - %v", err)
		return err
	}
	dstLink, err := netlink.LinkByName(cfg.dstName)
	if err != nil {
		logrus.Errorf("Failed to get destination link from name - %v", err)
		return err
	}

	/* Inject each end in its corresponding client/endpoint namespace */
	if err = netlink.LinkSetNsFd(srcLink, int(srcNsHandle)); err != nil {
		logrus.Errorf("Failed to inject the VETH end in the source namespace - %v", err)
		return err
	}
	if err = netlink.LinkSetNsFd(dstLink, int(dstNsHandle)); err != nil {
		logrus.Errorf("Failed to inject the VETH end in the destination namespace - %v", err)
		return err
	}
	return nil
}
