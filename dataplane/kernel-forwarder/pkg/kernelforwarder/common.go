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
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
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
	id         string
	srcNsPath  string
	dstNsPath  string
	srcName    string
	dstName    string
	srcIP      string
	dstIP      string
	srcIPVXLAN net.IP
	dstIPVXLAN net.IP
	srcRoutes  []*connectioncontext.Route
	dstRoutes  []*connectioncontext.Route
	neighbors  []*connectioncontext.IpNeighbor
	vni        int
}

// setupLinkInNs is responsible for configuring an interface inside a given namespace - assigns IP address, routes, etc.
func setupLinkInNs(containerNs netns.NsHandle, ifaceName, ifaceIP string, routes []*connectioncontext.Route, neighbors []*connectioncontext.IpNeighbor, inject bool) error {
	if inject {
		/* 1. Get a link object for the interface */
		ifaceLink, err := netlink.LinkByName(ifaceName)
		if err != nil {
			logrus.Errorf("common: failed to get link for %q - %v", ifaceName, err)
			return err
		}
		/* 2. Inject the interface into the desired container namespace */
		if err = netlink.LinkSetNsFd(ifaceLink, int(containerNs)); err != nil {
			logrus.Errorf("common: failed to inject %q in namespace - %v", ifaceName, err)
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
			logrus.Error("common: error when closing:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("common: failed to get current namespace: %v", err)
		return err
	}
	/* 5. Switch to the new namespace */
	if err = netns.Set(containerNs); err != nil {
		logrus.Errorf("common: failed to switch to container namespace: %v", err)
		return err
	}
	/* 6. Get a link for the interface name */
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		logrus.Errorf("common: failed to lookup %q, %v", ifaceName, err)
		return err
	}
	if inject {
		var addr *netlink.Addr
		/* 7. Parse the IP address */
		addr, err = netlink.ParseAddr(ifaceIP)
		if err != nil {
			logrus.Errorf("common: failed to parse IP %q: %v", ifaceIP, err)
			return err
		}
		/* 8. Set IP address */
		if err = netlink.AddrAdd(link, addr); err != nil {
			logrus.Errorf("common: failed to set IP %q: %v", ifaceIP, err)
			return err
		}
		/* 9. Bring the interface UP */
		if err = netlink.LinkSetUp(link); err != nil {
			logrus.Errorf("common: failed to bring %q up: %v", ifaceName, err)
			return err
		}
		/* 10. Add routes */
		if err = addRoutes(link, addr, routes); err != nil {
			logrus.Error("common: failed adding routes:", err)
		}
		/* 11. Add neighbors - applicable only for source side */
		if err = addNeighbors(link, neighbors); err != nil {
			logrus.Error("common: failed adding neighbors:", err)
		}
	} else {
		/* 7. Bring the interface DOWN */
		if err = netlink.LinkSetDown(link); err != nil {
			logrus.Errorf("common: failed to bring %q down: %v", ifaceName, err)
			return err
		}
		/* 8. Inject the interface back into the host namespace */
		if err = netlink.LinkSetNsFd(link, int(currentNs)); err != nil {
			logrus.Errorf("common: failed to inject %q back to host namespace - %v", ifaceName, err)
			return err
		}
	}
	/* Switch back to the original namespace */
	if err = netns.Set(currentNs); err != nil {
		logrus.Errorf("common: failed to switch back to original namespace: %v", err)
		return err
	}
	return nil
}

//nolint
func newConnectionConfig(crossConnect *crossconnect.CrossConnect, connType uint8) (*connectionConfig, error) {
	switch connType {
	case cLOCAL:
		srcNsPath, err := crossConnect.GetLocalSource().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("common: failed to get source namespace path - %v", err)
			return nil, err
		}
		dstNsPath, err := crossConnect.GetLocalDestination().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("common: failed to get destination namespace path - %v", err)
			return nil, err
		}
		return &connectionConfig{
			id:        crossConnect.GetId(),
			srcNsPath: srcNsPath,
			dstNsPath: dstNsPath,
			srcName:   crossConnect.GetLocalSource().GetMechanism().GetParameters()[local.InterfaceNameKey],
			dstName:   crossConnect.GetLocalDestination().GetMechanism().GetParameters()[local.InterfaceNameKey],
			srcIP:     crossConnect.GetLocalSource().GetContext().GetIpContext().GetSrcIpAddr(),
			dstIP:     crossConnect.GetLocalSource().GetContext().GetIpContext().GetDstIpAddr(),
			srcRoutes: crossConnect.GetLocalSource().GetContext().GetIpContext().GetDstRoutes(),
			dstRoutes: crossConnect.GetLocalDestination().GetContext().GetIpContext().GetSrcRoutes(),
			neighbors: crossConnect.GetLocalSource().GetContext().GetIpContext().GetIpNeighbors(),
		}, nil
	case cINCOMING:
		dstNsPath, err := crossConnect.GetLocalDestination().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("common: failed to get destination namespace path - %v", err)
			return nil, err
		}
		vni, _ := strconv.Atoi(crossConnect.GetRemoteSource().GetMechanism().GetParameters()[remote.VXLANVNI])
		return &connectionConfig{
			id:         crossConnect.GetId(),
			dstNsPath:  dstNsPath,
			dstName:    crossConnect.GetLocalDestination().GetMechanism().GetParameters()[local.InterfaceNameKey],
			dstIP:      crossConnect.GetLocalDestination().GetContext().GetIpContext().GetDstIpAddr(),
			dstRoutes:  crossConnect.GetLocalDestination().GetContext().GetIpContext().GetSrcRoutes(),
			neighbors:  nil,
			srcIPVXLAN: net.ParseIP(crossConnect.GetRemoteSource().GetMechanism().GetParameters()[remote.VXLANSrcIP]),
			dstIPVXLAN: net.ParseIP(crossConnect.GetRemoteSource().GetMechanism().GetParameters()[remote.VXLANDstIP]),
			vni:        vni,
		}, nil
	case cOUTGOING:
		srcNsPath, err := crossConnect.GetLocalSource().GetMechanism().NetNsFileName()
		if err != nil {
			logrus.Errorf("common: failed to get destination namespace path - %v", err)
			return nil, err
		}
		vni, _ := strconv.Atoi(crossConnect.GetRemoteDestination().GetMechanism().GetParameters()[remote.VXLANVNI])
		return &connectionConfig{
			id:         crossConnect.GetId(),
			srcNsPath:  srcNsPath,
			srcName:    crossConnect.GetLocalSource().GetMechanism().GetParameters()[local.InterfaceNameKey],
			srcIP:      crossConnect.GetLocalSource().GetContext().GetIpContext().GetSrcIpAddr(),
			srcRoutes:  crossConnect.GetLocalSource().GetContext().GetIpContext().GetDstRoutes(),
			neighbors:  crossConnect.GetLocalSource().GetContext().GetIpContext().GetIpNeighbors(),
			srcIPVXLAN: net.ParseIP(crossConnect.GetRemoteDestination().GetMechanism().GetParameters()[remote.VXLANSrcIP]),
			dstIPVXLAN: net.ParseIP(crossConnect.GetRemoteDestination().GetMechanism().GetParameters()[remote.VXLANDstIP]),
			vni:        vni,
		}, nil
	default:
		logrus.Error("common: connection configuration: invalid connection type")
		return nil, fmt.Errorf("common: invalid connection type")
	}
}

// addRoutes adds routes
func addRoutes(link netlink.Link, addr *netlink.Addr, routes []*connectioncontext.Route) error {
	for _, route := range routes {
		_, routeNet, err := net.ParseCIDR(route.GetPrefix())
		if err != nil {
			logrus.Error("common: failed parsing route CIDR:", err)
			return err
		}
		route := netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst: &net.IPNet{
				IP:   routeNet.IP,
				Mask: routeNet.Mask,
			},
			Src: addr.IP,
		}
		if err = netlink.RouteAdd(&route); err != nil {
			logrus.Error("common: failed adding routes:", err)
			return err
		}
	}
	return nil
}

// addNeighbors adds neighbors
func addNeighbors(link netlink.Link, neighbors []*connectioncontext.IpNeighbor) error {
	for _, neighbor := range neighbors {
		mac, err := net.ParseMAC(neighbor.GetHardwareAddress())
		if err != nil {
			logrus.Error("common: failed parsing the MAC address for IP neighbors:", err)
			return err
		}
		neigh := netlink.Neigh{
			LinkIndex:    link.Attrs().Index,
			State:        0x02, // netlink.NUD_REACHABLE, // the constant is somehow not being found in the package in case of using a darwin based machine
			IP:           net.ParseIP(neighbor.GetIp()),
			HardwareAddr: mac,
		}
		if err = netlink.NeighAdd(&neigh); err != nil {
			logrus.Error("common: failed adding neighbor:", err)
			return err
		}
	}
	return nil
}
