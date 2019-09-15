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

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

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
	defer func() {
		if err = srcNsHandle.Close(); err != nil {
			logrus.Error("error when closing:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("failed to get source namespace handler from path - %v", err)
		return err
	}
	dstNsHandle, err := netns.GetFromPath(cfg.dstNsPath)
	defer func() {
		if err = dstNsHandle.Close(); err != nil {
			logrus.Error("error when closing:", err)
		}
	}()
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
	if err = setupLinkInNs(srcNsHandle, cfg.srcName, cfg.srcIP, cfg.srcMac, cfg.srcRoutes, cfg.neighbors, true); err != nil {
		logrus.Errorf("failed to setup container interface %q: %v", cfg.srcName, err)
		return err
	}

	/* 5. Setup interface - destination namespace */
	if err = setupLinkInNs(dstNsHandle, cfg.dstName, cfg.dstIP, cfg.dstMac, cfg.dstRoutes, nil, true); err != nil {
		logrus.Errorf("failed to setup container interface %q: %v", cfg.dstName, err)
		return err
	}
	return nil
}

func deleteLocalConnection(cfg *connectionConfig) error {
	logrus.Info("Delete local connection...")
	/* 1. Get handlers for source and destination namespaces */
	srcNsHandle, err := netns.GetFromPath(cfg.srcNsPath)
	defer func() {
		if err = srcNsHandle.Close(); err != nil {
			logrus.Error("error when closing:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("failed to get source namespace handler from path - %v", err)
		return err
	}
	dstNsHandle, err := netns.GetFromPath(cfg.dstNsPath)
	defer func() {
		if err = dstNsHandle.Close(); err != nil {
			logrus.Error("error when closing:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("failed to get destination namespace handler from path - %v", err)
		return err
	}

	/* 2. Extract the interface - source namespace */
	if err = setupLinkInNs(srcNsHandle, cfg.srcName, cfg.srcMac, cfg.srcIP, nil, nil, false); err != nil {
		logrus.Errorf("failed to setup container interface %q: %v", cfg.srcName, err)
		return err
	}

	/* 3. Extract the interface - destination namespace */
	if err = setupLinkInNs(dstNsHandle, cfg.dstName, cfg.dstMac, cfg.dstIP, nil, nil, false); err != nil {
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
