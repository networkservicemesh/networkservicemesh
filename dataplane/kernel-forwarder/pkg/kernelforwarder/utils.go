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
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"os"
	"runtime"
)

type KernelConnectionConfig struct {
	srcNsPath string
	dstNsPath string
	srcName   string
	dstName   string
	srcIP     string
	dstIP     string
}

func handleKernelConnectionLocal(crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	/* Create a connection */
	if connect {
		/* 1. Get the connection configuration */
		cfg, err := getConnectionConfig(crossConnect)
		if err != nil {
			logrus.Errorf("Failed to get the configuration for local connection - %v", err)
			return crossConnect, err
		}
		/* 2. Get namespace handlers from their path - source and destination */
		srcNsHandle, err := netns.GetFromPath(cfg.srcNsPath)
		defer srcNsHandle.Close()
		if err != nil {
			logrus.Errorf("Failed to get source namespace handler from path - %v", err)
			return crossConnect, err
		}
		dstNsHandle, err := netns.GetFromPath(cfg.dstNsPath)
		defer dstNsHandle.Close()
		if err != nil {
			logrus.Errorf("Failed to get destination namespace handler from path - %v", err)
			return crossConnect, err
		}
		/* 3. Create a VETH pair and inject each end in the corresponding namespace */
		if err = createVETH(cfg, srcNsHandle, dstNsHandle); err != nil {
			logrus.Errorf("Failed to create the VETH pair - %v", err)
			return crossConnect, err
		}
		/* 4. Bring up and configure each pair end with its IP address */
		setupVETHEnd(srcNsHandle, cfg.srcName, cfg.srcIP)
		setupVETHEnd(dstNsHandle, cfg.dstName, cfg.dstIP)
	}
	/* Delete a connection */
	return crossConnect, nil
}

func handleKernelConnectionRemote(ctx context.Context, crossConnect *crossconnect.CrossConnect, connect bool) (*crossconnect.CrossConnect, error) {
	logrus.Errorf("Remote connection is not supported yet.")
	return crossConnect, nil
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
		srcName:   crossConnect.GetLocalSource().GetMechanism().GetParameters()[local.InterfaceNameKey],
		dstName:   crossConnect.GetLocalDestination().GetMechanism().GetParameters()[local.InterfaceNameKey],
		srcIP:     crossConnect.GetLocalSource().GetContext().GetSrcIpAddr(),
		dstIP:     crossConnect.GetLocalSource().GetContext().GetDstIpAddr(),
	}, nil
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

func createVETH(cfg *KernelConnectionConfig, srcNsHandle, dstNsHandle netns.NsHandle) error {
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

func setDataplaneConfigBase(v *KernelForwarder, common *common.DataplaneConfigBase) {
	var ok bool
	v.common = common
	v.common.Name, ok = os.LookupEnv(DataplaneNameKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneNameKey, DataplaneNameDefault)
		v.common.Name = DataplaneNameDefault
	}

	logrus.Infof("Starting dataplane - %s", v.common.Name)
	v.common.DataplaneSocket, ok = os.LookupEnv(DataplaneSocketKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketKey, DataplaneSocketDefault)
		v.common.DataplaneSocket = DataplaneSocketDefault
	}
	logrus.Infof("DataplaneSocket: %s", v.common.DataplaneSocket)

	v.common.DataplaneSocketType, ok = os.LookupEnv(DataplaneSocketTypeKey)
	if !ok {
		logrus.Infof("%s not set, using default %s", DataplaneSocketTypeKey, DataplaneSocketTypeDefault)
		v.common.DataplaneSocketType = DataplaneSocketTypeDefault
	}
	logrus.Infof("DataplaneSocketType: %s", v.common.DataplaneSocketType)
}

func setDataplaneConfigKernelForwarder(v *KernelForwarder, monitor monitor_crossconnect.MonitorServer) {
	var err error

	v.monitor = monitor

	srcIPStr, ok := os.LookupEnv(SrcIPEnvKey)
	if !ok {
		logrus.Fatalf("Env variable %s must be set to valid srcIP for use for tunnels from this Pod.  Consider using downward API to do so.", SrcIPEnvKey)
		common.SetSrcIPFailed()
	}
	v.srcIP = net.ParseIP(srcIPStr)
	if v.srcIP == nil {
		logrus.Fatalf("Env variable %s must be set to a valid IP address, was set to %s", SrcIPEnvKey, srcIPStr)
		common.SetValidIPFailed()
	}
	v.egressInterface, err = common.NewEgressInterface(v.srcIP)
	if err != nil {
		logrus.Fatalf("Unable to find egress Interface: %s", err)
		common.SetNewEgressIFFailed()
	}
	logrus.Infof("SrcIP: %s, IfaceName: %s, SrcIPNet: %s", v.srcIP, v.egressInterface.Name(), v.egressInterface.SrcIPNet())

	err = tools.SocketCleanup(v.common.DataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error cleaning up socket %s: %s", v.common.DataplaneSocket, err)
		common.SetSocketCleanFailed()
	}
	v.updateCh = make(chan *Mechanisms, 1)
	v.mechanisms = &Mechanisms{
		localMechanisms: []*local.Mechanism{
			{
				Type: local.MechanismType_KERNEL_INTERFACE,
			},
		},
		remoteMechanisms: []*remote.Mechanism{
			{
				Type: remote.MechanismType_VXLAN,
				Parameters: map[string]string{
					remote.VXLANSrcIP: v.egressInterface.SrcIPNet().IP.String(),
				},
			},
		},
	}
}
