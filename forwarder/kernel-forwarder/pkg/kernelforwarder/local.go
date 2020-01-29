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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/monitoring"
)

// handleLocalConnection either creates or deletes a local connection - same host
func (k *KernelForwarder) handleLocalConnection(crossConnect *crossconnect.CrossConnect, connect bool) (map[string]monitoring.Device, error) {
	logrus.Info("local: connection type - local source/local destination")
	var devices map[string]monitoring.Device
	var err error
	if connect {
		/* 2. Create a connection */
		devices, err = k.createLocalConnection(crossConnect)
		if err != nil {
			logrus.Errorf("local: failed to create connection - %v", err)
			devices = nil
		}
	} else {
		/* 3. Delete a connection */
		devices, err = k.deleteLocalConnection(crossConnect)
		if err != nil {
			logrus.Errorf("local: failed to delete connection - %v", err)
			devices = nil
		}
	}
	return devices, err
}

// createLocalConnection handles creating a local connection
func (k *KernelForwarder) createLocalConnection(crossConnect *crossconnect.CrossConnect) (map[string]monitoring.Device, error) {
	logrus.Info("local: creating connection...")
	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	srcName := crossConnect.GetSource().GetMechanism().GetParameters()[common.InterfaceNameKey]
	dstName := crossConnect.GetDestination().GetMechanism().GetParameters()[common.InterfaceNameKey]
	var srcNetNsInode string
	var dstNetNsInode string
	var err error

	if err = k.localConnect.CreateInterfaces(srcName, dstName); err != nil {
		logrus.Errorf("local: %v", err)
		return nil, err
	}

	if srcNetNsInode, err = SetupInterface(srcName, crossConnect.GetSource(), false); err != nil {
		return nil, err
	}

	crossConnect.GetDestination().GetContext().IpContext = crossConnect.GetSource().GetContext().GetIpContext()
	if dstNetNsInode, err = SetupInterface(dstName, crossConnect.GetDestination(), true); err != nil {
		return nil, err
	}

	logrus.Infof("local: creation completed for devices - source: %s, destination: %s", srcName, dstName)

	srcDevice := monitoring.Device{Name: srcName, XconName: "SRC-" + crossConnect.GetId()}
	dstDevice := monitoring.Device{Name: dstName, XconName: "DST-" + crossConnect.GetId()}
	return map[string]monitoring.Device{srcNetNsInode: srcDevice, dstNetNsInode: dstDevice}, nil
}

// deleteLocalConnection handles deleting a local connection
func (k *KernelForwarder) deleteLocalConnection(crossConnect *crossconnect.CrossConnect) (map[string]monitoring.Device, error) {
	logrus.Info("local: deleting connection...")
	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	srcName := crossConnect.GetSource().GetMechanism().GetParameters()[common.InterfaceNameKey]
	dstName := crossConnect.GetDestination().GetMechanism().GetParameters()[common.InterfaceNameKey]

	srcNetNsInode, srcErr := ClearInterfaceSetup(srcName, crossConnect.GetSource())
	dstNetNsInode, dstErr := ClearInterfaceSetup(dstName, crossConnect.GetDestination())

	err := k.localConnect.DeleteInterfaces(srcName)

	if srcErr != nil || dstErr != nil || err != nil {
		return nil, errors.Errorf("local: %v - %v", srcErr, dstErr)
	}

	logrus.Infof("local: deletion completed for devices - source: %s, destination: %s", srcName, dstName)
	srcDevice := monitoring.Device{Name: srcName, XconName: "SRC-" + crossConnect.GetId()}
	dstDevice := monitoring.Device{Name: dstName, XconName: "DST-" + crossConnect.GetId()}
	return map[string]monitoring.Device{srcNetNsInode: srcDevice, dstNetNsInode: dstDevice}, nil
}
