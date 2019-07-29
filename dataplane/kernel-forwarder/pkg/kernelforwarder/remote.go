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

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
)

func handleRemoteConnection(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	if crossConnect.GetRemoteSource().GetMechanism().GetType() == remote.MechanismType_VXLAN &&
		crossConnect.GetLocalDestination().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE {
		/* 1. Incoming remote connection */
		logrus.Info("Incoming connection - remote source/local destination")
		return handleConnection(egress, crossConnect, connect, cINCOMING)
	} else if crossConnect.GetLocalSource().GetMechanism().GetType() == local.MechanismType_KERNEL_INTERFACE &&
		crossConnect.GetRemoteDestination().GetMechanism().GetType() == remote.MechanismType_VXLAN {
		/* 2. Outgoing remote connection */
		logrus.Info("Outgoing connection - local source/remote destination")
		return handleConnection(egress, crossConnect, connect, cOUTGOING)
	}
	logrus.Errorf("invalid remote connection type")
	return crossConnect, fmt.Errorf("invalid remote connection type")
}

func handleConnection(egress common.EgressInterfaceType, crossConnect *crossconnect.CrossConnect, connect bool, direction uint8) (*crossconnect.CrossConnect, error) {
	/* 1. Get the connection configuration */
	cfg, err := newConnectionConfig(crossConnect, direction)
	if err != nil {
		logrus.Errorf("failed to get the configuration for remote connection - %v", err)
		return crossConnect, err
	}
	nsPath, name, ifaceIP, vxlanIP := modifyConfiguration(cfg, direction)
	if connect {
		/* 2. Create a connection */
		err = createRemoteConnection(nsPath, name, ifaceIP, egress.SrcIPNet().IP, vxlanIP, cfg.vni)
		if err != nil {
			logrus.Errorf("failed to create remote connection - %v", err)
		}
	} else {
		/* 3. Delete a connection */
		err = deleteRemoteConnection(nsPath, name)
		if err != nil {
			logrus.Errorf("failed to delete remote connection - %v", err)
		}
	}
	return crossConnect, err
}

func createRemoteConnection(nsPath, ifaceName, ifaceIP string, egressIP, remoteIP net.IP, vni int) error {
	logrus.Info("Creating remote connection...")
	/* 1. Get handler for container namespace */
	containerNs, err := netns.GetFromPath(nsPath)
	defer func() {
		if err = containerNs.Close(); err != nil {
			logrus.Error("error when closing:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("failed to get namespace handler from path - %v", err)
		return err
	}

	/* 2. Prepare interface - VXLAN */
	iface := newVXLAN(ifaceName, egressIP, remoteIP, vni)

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

func deleteRemoteConnection(nsPath, ifaceName string) error {
	logrus.Info("Deleting remote connection...")
	/* 1. Get handler for container namespace */
	containerNs, err := netns.GetFromPath(nsPath)
	defer func() {
		if err = containerNs.Close(); err != nil {
			logrus.Error("error when closing:", err)
		}
	}()
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

/* modifyConfiguration swaps the values based on the direction of the connection - incoming or outgoing */
func modifyConfiguration(cfg *connectionConfig, direction uint8) (string, string, string, net.IP) {
	if direction == cINCOMING {
		return cfg.dstNsPath, cfg.dstName, cfg.dstIP, cfg.srcIPVXLAN
	}
	return cfg.srcNsPath, cfg.srcName, cfg.srcIP, cfg.dstIPVXLAN

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
