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

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
)

// handleRemoteConnection handles remote connect/disconnect requests for either incoming or outgoing connections
func handleRemoteConnection(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool) (map[string]string, error) {
	var err error
	var devices map[string]string

	if crossConnect.GetRemoteSource().GetMechanism().GetType() == remote.MechanismType_VXLAN &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE {
		/* 1. Incoming remote connection */
		logrus.Info("remote: connection type - incoming - remote source/local destination")
		devices, err = handleConnection(egress, crossConnect, connect, cINCOMING)
	} else if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE &&
		crossConnect.GetRemoteDestination().GetMechanism().GetType() == remote.MechanismType_VXLAN {
		/* 2. Outgoing remote connection */
		logrus.Info("remote: connection type - outgoing - local source/remote destination")
		devices, err = handleConnection(egress, crossConnect, connect, cOUTGOING)
	} else {
		logrus.Errorf("remote: invalid connection type")
		return nil, fmt.Errorf("remote: invalid connection type")
	}
	return devices, err
}

// handleConnection process the request to either creating or deleting a connection
func handleConnection(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool, direction uint8) (map[string]string, error) {
	var devices map[string]string

	/* 1. Get the connection configuration */
	cfg, err := newConnectionConfig(crossConnect, direction)
	if err != nil {
		logrus.Errorf("remote: failed to get connection configuration - %v", err)
		return nil, err
	}

	nsPath, name, ifaceIP, vxlanIP, routes := modifyConfiguration(cfg, direction)

	if connect {
		/* 2. Create a connection */
		devices, err = createRemoteConnection(nsPath, name, ifaceIP, egress.SrcIPNet().IP, vxlanIP, cfg.vni, routes, cfg.neighbors)
		if err != nil {
			logrus.Errorf("remote: failed to create connection - %v", err)
			devices = nil
		}
	} else {
		/* 3. Delete a connection */
		devices, err = deleteRemoteConnection(nsPath, name)
		if err != nil {
			logrus.Errorf("remote: failed to delete connection - %v", err)
			devices = nil
		}
	}
	return devices, err
}

// createRemoteConnection handler for creating a remote connection
func createRemoteConnection(nsPath, ifaceName, ifaceIP string, egressIP, remoteIP net.IP, vni int, routes []*connectioncontext.Route, neighbors []*connectioncontext.IpNeighbor) (map[string]string, error) {
	logrus.Info("remote: creating connection")
	/* 1. Get handler for container namespace */
	containerNs, err := netns.GetFromPath(nsPath)
	defer func() {
		if err = containerNs.Close(); err != nil {
			logrus.Error("remote: error when closing requested namespace:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("remote: failed to get requested namespace handler from path - %v", err)
		return nil, err
	}

	/* 2. Prepare interface - VXLAN */
	iface := newVXLAN(ifaceName, egressIP, remoteIP, vni)

	/* 3. Create interface - host namespace */
	if err = netlink.LinkAdd(iface); err != nil {
		logrus.Errorf("remote: failed to create VXLAN interface - %v", err)
	}

	/* 4. Setup interface */
	if err = setupLinkInNs(containerNs, ifaceName, ifaceIP, routes, neighbors, true); err != nil {
		logrus.Errorf("remote: failed to setup interface - destination - %q: %v", ifaceName, err)
		return nil, err
	}
	logrus.Infof("remote: creation completed for device - %s", ifaceName)
	return map[string]string{nsPath: ifaceName}, nil
}

// deleteRemoteConnection handler for deleting a remote connection
func deleteRemoteConnection(nsPath, ifaceName string) (map[string]string, error) {
	logrus.Info("remote: deleting connection")
	/* 1. Get handler for container namespace */
	containerNs, err := netns.GetFromPath(nsPath)
	defer func() {
		if err = containerNs.Close(); err != nil {
			logrus.Error("remote: error when closing requested namespace:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("remote: failed to get requested namespace handler from path - %v", err)
		return nil, err
	}

	/* 2. Setup interface */
	if err = setupLinkInNs(containerNs, ifaceName, "", nil, nil, false); err != nil {
		logrus.Errorf("remote: failed to setup interface - destination -  %q: %v", ifaceName, err)
		return nil, err
	}

	/* 3. Get a link object for the interface */
	ifaceLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		logrus.Errorf("remote: failed to get link for %q - %v", ifaceName, err)
		return nil, err
	}

	/* 4. Delete the VXLAN interface - host namespace */
	if err := netlink.LinkDel(ifaceLink); err != nil {
		logrus.Errorf("remote: failed to delete VXLAN interface - %v", err)
		return nil, err
	}
	logrus.Infof("remote: deletion completed for device - %s", ifaceName)
	return map[string]string{nsPath: ifaceName}, nil
}

// modifyConfiguration swaps the values based on the direction of the connection - incoming or outgoing
func modifyConfiguration(cfg *connectionConfig, direction uint8) (string, string, string, net.IP, []*connectioncontext.Route) {
	if direction == cINCOMING {
		return cfg.dstNsPath, cfg.dstName, cfg.dstIP, cfg.srcIPVXLAN, cfg.dstRoutes
	}
	return cfg.srcNsPath, cfg.srcName, cfg.srcIP, cfg.dstIPVXLAN, cfg.srcRoutes
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
