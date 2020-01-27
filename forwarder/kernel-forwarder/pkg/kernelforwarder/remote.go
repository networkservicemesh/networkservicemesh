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

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	common2 "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	. "github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/kernelforwarder/local"
	. "github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/kernelforwarder/remote"
	"github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/monitoring"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// handleRemoteConnection handles remote connect/disconnect requests for either incoming or outgoing connections
func handleRemoteConnection(crossConnect *crossconnect.CrossConnect, connect bool) (map[string]monitoring.Device, error) {
	if crossConnect.GetSource().IsRemote() && !crossConnect.GetDestination().IsRemote() {
		/* 1. Incoming remote connection */
		logrus.Info("remote: connection type - remote source/local destination - incoming")
		return handleConnection(crossConnect.GetId(), crossConnect.GetDestination(), crossConnect.GetSource(), connect, INCOMING)
	} else if !crossConnect.GetSource().IsRemote() && crossConnect.GetDestination().IsRemote(){
		/* 2. Outgoing remote connection */
		logrus.Info("remote: connection type - local source/remote destination - outgoing")
		return handleConnection(crossConnect.GetId(), crossConnect.GetSource(), crossConnect.GetDestination(), connect, OUTGOING)
	}
	err := errors.Errorf("remote: invalid connection type")
	logrus.Errorf("%+v", err)
	return nil, err
}

// handleConnection process the request to either creating or deleting a connection
func handleConnection(connId string, localConnection *connection.Connection, remoteConnection *connection.Connection, connect bool, direction uint8) (map[string]monitoring.Device, error) {
	var devices map[string]monitoring.Device
	var err error
	if connect {
		/* 2. Create a connection */
		devices, err = createRemoteConnection(connId, localConnection, remoteConnection, direction)
		if err != nil {
			logrus.Errorf("remote: failed to create connection - %v", err)
			devices = nil
		}
	} else {
		/* 3. Delete a connection */
		devices, err = deleteRemoteConnection(connId, localConnection, remoteConnection, direction)
		if err != nil {
			logrus.Errorf("remote: failed to delete connection - %v", err)
			devices = nil
		}
	}
	return devices, err
}

// createRemoteConnection handler for creating a remote connection
func createRemoteConnection(connId string, localConnection *connection.Connection, remoteConnection *connection.Connection, direction uint8) (map[string]monitoring.Device, error) {
	logrus.Info("remote: creating connection...")

	var xconName string
	if direction == INCOMING {
		xconName = "DST-" + connId
	} else {
		xconName = "SRC-" + connId
	}
	ifaceName := localConnection.GetMechanism().GetParameters()[common2.InterfaceNameKey]
	var nsInode string
	var err error

	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err = CreateRemoteInterface(xconName, remoteConnection, direction); err != nil {
		logrus.Errorf("remote: %v", err)
		return nil, err
	}

	if nsInode, err = CreateLocalInterface(ifaceName, localConnection); err != nil {
		logrus.Errorf("remote: %v", err)
		return nil, err
	}

	logrus.Infof("remote: creation completed for device - %s", ifaceName)
	return map[string]monitoring.Device{nsInode: {Name: ifaceName, XconName: xconName}}, nil
}

// deleteRemoteConnection handler for deleting a remote connection
func deleteRemoteConnection(connId string, localConnection *connection.Connection, remoteConnection *connection.Connection, direction uint8) (map[string]monitoring.Device, error) {
	logrus.Info("remote: deleting connection...")

	nsInode := localConnection.GetMechanism().GetParameters()[common2.NetNsInodeKey]
	ifaceName := localConnection.GetMechanism().GetParameters()[common2.InterfaceNameKey]
	var xconName string
	if direction == INCOMING {
		xconName = "DST-" + connId
	} else {
		xconName = "SRC-" + connId
	}

	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	nsInode, localErr := DeleteLocalInterface(ifaceName, localConnection)
	remoteErr := DeleteRemoteInterface(xconName)

	if localErr != nil || remoteErr != nil {
		logrus.Errorf("remote: %v - %v", localErr, remoteErr)
	}

	logrus.Infof("remote: deletion completed for device - %s", ifaceName)
	return map[string]monitoring.Device{nsInode: {Name: ifaceName, XconName: xconName}}, nil
}
