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
	"runtime"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"

	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/networkservicemesh/dataplane/kernel-forwarder/pkg/monitoring"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
)

// handleRemoteConnection handles remote connect/disconnect requests for either incoming or outgoing connections
func handleRemoteConnection(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool) (map[string]monitoring.Device, error) {
	if crossConnect.GetRemoteSource().GetMechanism().GetType() == remote.MechanismType_VXLAN &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE {
		/* 1. Incoming remote connection */
		logrus.Info("remote: connection type - incoming - remote source/local destination")
		return handleConnection(egress, crossConnect, connect, cINCOMING)
	} else if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE &&
		crossConnect.GetRemoteDestination().GetMechanism().GetType() == remote.MechanismType_VXLAN {
		/* 2. Outgoing remote connection */
		logrus.Info("remote: connection type - outgoing - local source/remote destination")
		return handleConnection(egress, crossConnect, connect, cOUTGOING)
	}
	logrus.Errorf("remote: invalid connection type")
	return nil, fmt.Errorf("remote: invalid connection type")
}

// handleConnection process the request to either creating or deleting a connection
func handleConnection(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool, direction uint8) (map[string]monitoring.Device, error) {
	var devices map[string]monitoring.Device

	/* 1. Get the connection configuration */
	cfg, err := newConnectionConfig(crossConnect, direction)
	if err != nil {
		logrus.Errorf("remote: failed to get connection configuration - %v", err)
		return nil, err
	}

	nsPath, name, ifaceIP, vxlanIP, routes, xconName := modifyConfiguration(cfg, direction)

	if connect {
		/* 2. Create a connection */
		devices, err = createRemoteConnection(nsPath, name, xconName, ifaceIP, egress.SrcIPNet().IP, vxlanIP, cfg.vni, routes, cfg.neighbors)
		if err != nil {
			logrus.Errorf("remote: failed to create connection - %v", err)
			devices = nil
		}
	} else {
		/* 3. Delete a connection */
		devices, err = deleteRemoteConnection(nsPath, name, xconName)
		if err != nil {
			logrus.Errorf("remote: failed to delete connection - %v", err)
			devices = nil
		}
	}
	return devices, err
}

// createRemoteConnection handler for creating a remote connection
func createRemoteConnection(nsInode, ifaceName, xconName, ifaceIP string, egressIP, remoteIP net.IP, vni int, routes []*connectioncontext.Route, neighbors []*connectioncontext.IpNeighbor) (map[string]monitoring.Device, error) {
	logrus.Info("remote: creating connection - ", nsInode, ifaceName, xconName, ifaceIP, vni)

	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	/* Get namespace handler - destination */
	dstHandle, err := fs.GetNsHandleFromInode(nsInode)
	if err != nil {
		logrus.Errorf("remote: failed to get destination namespace handle - %v", err)
		return nil, err
	}
	/* If successful, don't forget to close the handler upon exit */
	defer func() {
		if err = dstHandle.Close(); err != nil {
			logrus.Error("remote: error when closing destination handle: ", err)
		}
		logrus.Info("remote: closed destination handle: ", dstHandle, nsInode)
	}()
	logrus.Info("remote: opened destination handle: ", dstHandle, nsInode)

	/* Create interface - host namespace */
	if err = netlink.LinkAdd(newVXLAN(ifaceName, egressIP, remoteIP, vni)); err != nil {
		logrus.Errorf("remote: failed to create VXLAN interface - %v", err)
		return nil, err
	}

	/* Setup interface - inject from host to destination namespace */
	if err = setupLinkInNs(dstHandle, ifaceName, ifaceIP, routes, neighbors, true); err != nil {
		logrus.Errorf("remote: failed to setup interface - destination - %q: %v", ifaceName, err)
		return nil, err
	}
	logrus.Infof("remote: creation completed for device - %s", ifaceName)
	return map[string]monitoring.Device{nsInode: monitoring.Device{Name: ifaceName, XconName: xconName}}, nil
}

// deleteRemoteConnection handler for deleting a remote connection
func deleteRemoteConnection(nsInode, ifaceName, xconName string) (map[string]monitoring.Device, error) {
	logrus.Info("remote: deleting connection")

	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	/* Get namespace handler - destination */
	dstHandle, err := fs.GetNsHandleFromInode(nsInode)
	if err != nil {
		logrus.Errorf("remote: failed to get destination namespace handle - %v", err)
		return nil, err
	}
	/* If successful, don't forget to close the handler upon exit */
	defer func() {
		if err = dstHandle.Close(); err != nil {
			logrus.Error("remote: error when closing destination handle: ", err)
		}
		logrus.Info("remote: closed destination handle: ", dstHandle, nsInode)
	}()
	logrus.Info("remote: opened destination handle: ", dstHandle, nsInode)

	/* Setup interface - extract from destination to host namespace */
	if err = setupLinkInNs(dstHandle, ifaceName, "", nil, nil, false); err != nil {
		logrus.Errorf("remote: failed to setup interface - destination -  %q: %v", ifaceName, err)
		return nil, err
	}

	/* Get a link object for interface */
	ifaceLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		logrus.Errorf("remote: failed to get link for %q - %v", ifaceName, err)
		return nil, err
	}

	/* Delete the VXLAN interface - host namespace */
	if err := netlink.LinkDel(ifaceLink); err != nil {
		logrus.Errorf("remote: failed to delete VXLAN interface - %v", err)
		return nil, err
	}
	logrus.Infof("remote: deletion completed for device - %s", ifaceName)
	return map[string]monitoring.Device{nsInode: monitoring.Device{Name: ifaceName, XconName: xconName}}, nil
}

// modifyConfiguration swaps the values based on the direction of the connection - incoming or outgoing
func modifyConfiguration(cfg *connectionConfig, direction uint8) (string, string, string, net.IP, []*connectioncontext.Route, string) {
	if direction == cINCOMING {
		return cfg.dstNetNsInode, cfg.dstName, cfg.dstIP, cfg.srcIPVXLAN, cfg.dstRoutes, "DST-" + cfg.id
	}
	return cfg.srcNetNsInode, cfg.srcName, cfg.srcIP, cfg.dstIPVXLAN, cfg.srcRoutes, "SRC-" + cfg.id
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
