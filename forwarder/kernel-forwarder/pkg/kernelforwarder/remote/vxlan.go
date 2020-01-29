// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
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

package remote

import (
	"net"
	"strconv"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
)

// CreateVXLANInterface creates a VXLAN interface
func (c *Connect) createVXLANInterface(ifaceName string, remoteConnection *connection.Connection, direction uint8) error {
	/* Create interface - host namespace */
	srcIP := net.ParseIP(remoteConnection.GetMechanism().GetParameters()[vxlan.SrcIP])
	dstIP := net.ParseIP(remoteConnection.GetMechanism().GetParameters()[vxlan.DstIP])
	vni, _ := strconv.Atoi(remoteConnection.GetMechanism().GetParameters()[vxlan.VNI])

	var localIP net.IP
	var remoteIP net.IP
	if direction == INCOMING {
		localIP = dstIP
		remoteIP = srcIP
	} else {
		localIP = srcIP
		remoteIP = dstIP
	}

	if err := netlink.LinkAdd(newVXLAN(ifaceName, localIP, remoteIP, vni)); err != nil {
		return errors.Wrapf(err, "failed to create VXLAN interface")
	}
	return nil
}

func (c *Connect) deleteVXLANInterface(ifaceName string) error {
	/* Get a link object for interface */
	ifaceLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return errors.Errorf("failed to get link for %q - %v", ifaceName, err)
	}

	/* Delete the VXLAN interface - host namespace */
	if err = netlink.LinkDel(ifaceLink); err != nil {
		return errors.Errorf("failed to delete VXLAN interface - %v", err)
	}

	return nil
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
