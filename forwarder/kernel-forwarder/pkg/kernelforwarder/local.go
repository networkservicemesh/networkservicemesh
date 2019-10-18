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

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/monitoring"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// handleLocalConnection either creates or deletes a local connection - same host
func handleLocalConnection(crossConnect *crossconnect.CrossConnect, connect bool) (map[string]monitoring.Device, error) {
	logrus.Info("local: connection type - local source/local destination")
	var devices map[string]monitoring.Device
	/* 1. Get the connection configuration */
	cfg, err := newConnectionConfig(crossConnect, cLOCAL)
	if err != nil {
		logrus.Errorf("local: failed to get connection configuration - %v", err)
		return nil, err
	}
	if connect {
		/* 2. Create a connection */
		devices, err = createLocalConnection(cfg)
		if err != nil {
			logrus.Errorf("local: failed to create connection - %v", err)
			devices = nil
		}
	} else {
		/* 3. Delete a connection */
		devices, err = deleteLocalConnection(cfg)
		if err != nil {
			logrus.Errorf("local: failed to delete connection - %v", err)
			devices = nil
		}
	}
	return devices, err
}

// createLocalConnection handles creating a local connection
func createLocalConnection(cfg *connectionConfig) (map[string]monitoring.Device, error) {
	logrus.Info("local: creating connection...")
	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	/* Get namespace handler - source */
	srcNsHandle, err := fs.GetNsHandleFromInode(cfg.srcNetNsInode)
	if err != nil {
		logrus.Errorf("local: failed to get source namespace handle - %v", err)
		return nil, err
	}
	/* If successful, don't forget to close the handler upon exit */
	defer func() {
		if err = srcNsHandle.Close(); err != nil {
			logrus.Error("local: error when closing source handle: ", err)
		}
		logrus.Debug("local: closed source handle: ", srcNsHandle, cfg.srcNetNsInode)
	}()
	logrus.Debug("local: opened source handle: ", srcNsHandle, cfg.srcNetNsInode)

	/* Get namespace handler - destination */
	dstNsHandle, err := fs.GetNsHandleFromInode(cfg.dstNetNsInode)
	if err != nil {
		logrus.Errorf("local: failed to get destination namespace handle - %v", err)
		return nil, err
	}
	defer func() {
		if err = dstNsHandle.Close(); err != nil {
			logrus.Error("local: error when closing destination handle: ", err)
		}
		logrus.Debug("local: closed destination handle: ", dstNsHandle, cfg.dstNetNsInode)
	}()
	logrus.Debug("local: opened destination handle: ", dstNsHandle, cfg.dstNetNsInode)

	/* Create the VETH pair - host namespace */
	if err = netlink.LinkAdd(newVETH(cfg.srcName, cfg.dstName)); err != nil {
		logrus.Errorf("local: failed to create VETH pair - %v", err)
		return nil, err
	}

	/* Setup interface - source namespace */
	if err = setupLinkInNs(srcNsHandle, cfg.srcName, cfg.srcIP, cfg.srcRoutes, cfg.neighbors, true); err != nil {
		logrus.Errorf("local: failed to setup interface - source - %q: %v", cfg.srcName, err)
		return nil, err
	}

	/* Setup interface - destination namespace */
	if err = setupLinkInNs(dstNsHandle, cfg.dstName, cfg.dstIP, cfg.dstRoutes, nil, true); err != nil {
		logrus.Errorf("local: failed to setup interface - destination - %q: %v", cfg.dstName, err)
		return nil, err
	}

	logrus.Infof("local: creation completed for devices - source: %s, destination: %s", cfg.srcName, cfg.dstName)
	srcDevice := monitoring.Device{Name: cfg.srcName, XconName: "SRC-" + cfg.id}
	dstDevice := monitoring.Device{Name: cfg.dstName, XconName: "DST-" + cfg.id}
	return map[string]monitoring.Device{cfg.srcNetNsInode: srcDevice, cfg.dstNetNsInode: dstDevice}, nil
}

// deleteLocalConnection handles deleting a local connection
func deleteLocalConnection(cfg *connectionConfig) (map[string]monitoring.Device, error) {
	logrus.Info("local: deleting connection...")
	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	/* Get namespace handler - source */
	srcNsHandle, err := fs.GetNsHandleFromInode(cfg.srcNetNsInode)
	if err != nil {
		logrus.Errorf("local: failed to get source namespace handle - %v", err)
		return nil, err
	}
	/* If successful, don't forget to close the handler upon exit */
	defer func() {
		if err = srcNsHandle.Close(); err != nil {
			logrus.Error("local: error when closing source handle: ", err)
		}
		logrus.Debug("local: closed source handle: ", srcNsHandle, cfg.srcNetNsInode)
	}()
	logrus.Debug("local: opened source handle: ", srcNsHandle, cfg.srcNetNsInode)

	/* Get namespace handler - destination */
	dstNsHandle, err := fs.GetNsHandleFromInode(cfg.dstNetNsInode)
	if err != nil {
		logrus.Errorf("local: failed to get destination namespace handle - %v", err)
		return nil, err
	}
	defer func() {
		if err = dstNsHandle.Close(); err != nil {
			logrus.Error("local: error when closing destination handle: ", err)
		}
		logrus.Debug("local: closed destination handle: ", dstNsHandle, cfg.dstNetNsInode)
	}()
	logrus.Debug("local: opened destination handle: ", dstNsHandle, cfg.dstNetNsInode)

	/* Extract interface - source namespace */
	if err = setupLinkInNs(srcNsHandle, cfg.srcName, cfg.srcIP, nil, nil, false); err != nil {
		logrus.Errorf("local: failed to extract interface - source - %q: %v", cfg.srcName, err)
		return nil, err
	}

	/* Extract interface - destination namespace */
	if err = setupLinkInNs(dstNsHandle, cfg.dstName, cfg.dstIP, nil, nil, false); err != nil {
		logrus.Errorf("local: failed to extract interface - destination - %q: %v", cfg.dstName, err)
		return nil, err
	}

	/* Get a link object for the interface */
	ifaceLink, err := netlink.LinkByName(cfg.srcName)
	if err != nil {
		logrus.Errorf("local: failed to get link for %q - %v", cfg.srcName, err)
		return nil, err
	}

	/* Delete the VETH pair - host namespace */
	if err := netlink.LinkDel(ifaceLink); err != nil {
		logrus.Errorf("local: failed to delete the VETH pair - %v", err)
		return nil, err
	}

	logrus.Infof("local: deletion completed for devices - source: %s, destination: %s", cfg.srcName, cfg.dstName)
	srcDevice := monitoring.Device{Name: cfg.srcName, XconName: "SRC-" + cfg.id}
	dstDevice := monitoring.Device{Name: cfg.dstName, XconName: "DST-" + cfg.id}
	return map[string]monitoring.Device{cfg.srcNetNsInode: srcDevice, cfg.dstNetNsInode: dstDevice}, nil
}

// newVETH returns a VETH interface instance
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
