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
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"

	"net"
	"strconv"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"

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
	id            string
	srcNetNsInode string
	dstNetNsInode string
	srcName       string
	dstName       string
	srcIP         string
	dstIP         string
	srcIPVXLAN    net.IP
	dstIPVXLAN    net.IP
	srcRoutes     []*networkservice.Route
	dstRoutes     []*networkservice.Route
	neighbors     []*networkservice.IpNeighbor
	vni           int
}

// setupLinkInNs is responsible for configuring an interface inside a given namespace - assigns IP address, routes, etc.
func setupLinkInNs(containerNs netns.NsHandle, ifaceName, ifaceIP string, routes []*networkservice.Route, neighbors []*networkservice.IpNeighbor, inject bool) error {
	if inject {
		/* Get a link object for the interface */
		ifaceLink, err := netlink.LinkByName(ifaceName)
		if err != nil {
			logrus.Errorf("common: failed to get link for %q - %v", ifaceName, err)
			return err
		}
		/* Inject the interface into the desired namespace */
		if err = netlink.LinkSetNsFd(ifaceLink, int(containerNs)); err != nil {
			logrus.Errorf("common: failed to inject %q in namespace - %v", ifaceName, err)
			return err
		}
	}
	/* Save current network namespace */
	hostNs, err := netns.Get()
	if err != nil {
		logrus.Errorf("common: failed getting host namespace: %v", err)
		return err
	}
	logrus.Debug("common: host namespace: ", hostNs)
	defer func() {
		if err = hostNs.Close(); err != nil {
			logrus.Error("common: failed closing host namespace handle: ", err)
		}
		logrus.Debug("common: closed host namespace handle: ", hostNs)
	}()

	/* Switch to the desired namespace */
	if err = netns.Set(containerNs); err != nil {
		logrus.Errorf("common: failed switching to desired namespace: %v", err)
		return err
	}
	logrus.Debug("common: switched to desired namespace: ", containerNs)

	/* Don't forget to switch back to the host namespace */
	defer func() {
		if err = netns.Set(hostNs); err != nil {
			logrus.Errorf("common: failed switching back to host namespace: %v", err)
		}
		logrus.Debug("common: switched back to host namespace: ", hostNs)
	}()

	/* Get a link for the interface name */
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		logrus.Errorf("common: failed to lookup %q, %v", ifaceName, err)
		return err
	}
	if inject {
		var addr *netlink.Addr
		/* Parse the IP address */
		addr, err = netlink.ParseAddr(ifaceIP)
		if err != nil {
			logrus.Errorf("common: failed to parse IP %q: %v", ifaceIP, err)
			return err
		}
		/* Set IP address */
		if err = netlink.AddrAdd(link, addr); err != nil {
			logrus.Errorf("common: failed to set IP %q: %v", ifaceIP, err)
			return err
		}
		/* Bring the interface UP */
		if err = netlink.LinkSetUp(link); err != nil {
			logrus.Errorf("common: failed to bring %q up: %v", ifaceName, err)
			return err
		}
		/* Add routes */
		if err = addRoutes(link, addr, routes); err != nil {
			logrus.Error("common: failed adding routes:", err)
			return err
		}
		/* Add neighbors - applicable only for source side */
		if err = addNeighbors(link, neighbors); err != nil {
			logrus.Error("common: failed adding neighbors:", err)
			return err
		}
	} else {
		/* Bring the interface DOWN */
		if err = netlink.LinkSetDown(link); err != nil {
			logrus.Errorf("common: failed to bring %q down: %v", ifaceName, err)
			return err
		}
		/* Inject the interface back to current namespace */
		if err = netlink.LinkSetNsFd(link, int(hostNs)); err != nil {
			logrus.Errorf("common: failed to inject %q back to host namespace - %v", ifaceName, err)
			return err
		}
	}
	return nil
}

//nolint
func newConnectionConfig(crossConnect *crossconnect.CrossConnect, connType uint8) (*connectionConfig, error) {
	switch connType {
	case cLOCAL:
		return &connectionConfig{
			id:            crossConnect.GetId(),
			srcNetNsInode: crossConnect.GetSource().GetMechanism().GetParameters()[common.NetNSInodeKey],
			dstNetNsInode: crossConnect.GetDestination().GetMechanism().GetParameters()[common.NetNSInodeKey],
			srcName:       crossConnect.GetSource().GetMechanism().GetParameters()[common.InterfaceNameKey],
			dstName:       crossConnect.GetDestination().GetMechanism().GetParameters()[common.InterfaceNameKey],
			srcIP:         crossConnect.GetSource().GetContext().GetIpContext().GetSrcIpAddr(),
			dstIP:         crossConnect.GetSource().GetContext().GetIpContext().GetDstIpAddr(),
			srcRoutes:     crossConnect.GetSource().GetContext().GetIpContext().GetDstRoutes(),
			dstRoutes:     crossConnect.GetDestination().GetContext().GetIpContext().GetSrcRoutes(),
			neighbors:     crossConnect.GetSource().GetContext().GetIpContext().GetIpNeighbors(),
		}, nil
	case cINCOMING:
		vni, _ := strconv.Atoi(crossConnect.GetSource().GetMechanism().GetParameters()[vxlan.VNI])
		return &connectionConfig{
			id:            crossConnect.GetId(),
			dstNetNsInode: crossConnect.GetDestination().GetMechanism().GetParameters()[common.NetNSInodeKey],
			dstName:       crossConnect.GetDestination().GetMechanism().GetParameters()[common.InterfaceNameKey],
			dstIP:         crossConnect.GetDestination().GetContext().GetIpContext().GetDstIpAddr(),
			dstRoutes:     crossConnect.GetDestination().GetContext().GetIpContext().GetSrcRoutes(),
			neighbors:     nil,
			srcIPVXLAN:    net.ParseIP(crossConnect.GetSource().GetMechanism().GetParameters()[vxlan.SrcIP]),
			dstIPVXLAN:    net.ParseIP(crossConnect.GetSource().GetMechanism().GetParameters()[vxlan.DstIP]),
			vni:           vni,
		}, nil
	case cOUTGOING:
		vni, _ := strconv.Atoi(crossConnect.GetDestination().GetMechanism().GetParameters()[vxlan.VNI])
		return &connectionConfig{
			id:            crossConnect.GetId(),
			srcNetNsInode: crossConnect.GetSource().GetMechanism().GetParameters()[common.NetNSInodeKey],
			srcName:       crossConnect.GetSource().GetMechanism().GetParameters()[common.InterfaceNameKey],
			srcIP:         crossConnect.GetSource().GetContext().GetIpContext().GetSrcIpAddr(),
			srcRoutes:     crossConnect.GetSource().GetContext().GetIpContext().GetDstRoutes(),
			neighbors:     crossConnect.GetSource().GetContext().GetIpContext().GetIpNeighbors(),
			srcIPVXLAN:    net.ParseIP(crossConnect.GetDestination().GetMechanism().GetParameters()[vxlan.SrcIP]),
			dstIPVXLAN:    net.ParseIP(crossConnect.GetDestination().GetMechanism().GetParameters()[vxlan.DstIP]),
			vni:           vni,
		}, nil
	default:
		logrus.Error("common: connection configuration: invalid connection type")
		return nil, errors.New("common: invalid connection type")
	}
}

// addRoutes adds routes
func addRoutes(link netlink.Link, addr *netlink.Addr, routes []*networkservice.Route) error {
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
func addNeighbors(link netlink.Link, neighbors []*networkservice.IpNeighbor) error {
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
