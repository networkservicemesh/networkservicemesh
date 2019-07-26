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
	/* VETH pairs are used only for local connections(same node), so we can use a larger MTU size as there's no multi-node connection */
	cVETHMTU    = 16000
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
	srcIPVXLAN net.IP
	dstIPVXLAN net.IP
	vni        int
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
	defer func() {
		if err = currentNs.Close(); err != nil {
			logrus.Error("error when closing:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("failed to get current namespace: %v", err)
		return err
	}
	/* 5. Switch to the new namespace */
	if err = netns.Set(containerNs); err != nil {
		logrus.Errorf("failed to switch to container namespace: %v", err)
		return err
	}
	defer func() {
		if err = containerNs.Close(); err != nil {
			logrus.Error("error when closing:", err)
		}
	}()
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
