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
	"net"
	"runtime"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netns"
	"github.com/vishvananda/netlink"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
)

type KernelConnectionConfig struct {
	srcNsPath string
	dstNsPath string
	srcName string
	dstName string
	srcIP string
	dstIP string
}

func getConnectionConfig(crossConnect *crossconnect.CrossConnect) (*KernelConnectionConfig, error) {
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
	return &KernelConnectionConfig{
		srcNsPath: srcNsPath,
		dstNsPath: dstNsPath,
		srcName: crossConnect.GetLocalSource().GetMechanism().GetParameters()[local.InterfaceNameKey],
		dstName: crossConnect.GetLocalDestination().GetMechanism().GetParameters()[local.InterfaceNameKey],
		srcIP: crossConnect.GetLocalSource().GetContext().GetSrcIpAddr(),
		dstIP: crossConnect.GetLocalSource().GetContext().GetDstIpAddr(),
	}, nil
}

func setupVETHEnd(nsHandle netns.NsHandle, ifName, addrIP string) {
	/* Lock the OS Thread so we don't accidentally switch namespaces */
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

func createVETH(cfg *KernelConnectionConfig, srcNsHandle, dstNsHandle netns.NsHandle) error {
	/* Initial VETH configuration */
	cfgVETH := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:  cfg.srcName,
			Flags: net.FlagUp,
			MTU:   1500,
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