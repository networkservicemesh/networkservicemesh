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

// Package local - controlling local mechanisms interfaces
package local

import (
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

const (
	/* VETH pairs are used only for local connections(same node), so we can use a larger MTU size as there's no multi-node connection */
	cVETHMTU = 16000
)

// Connect - struct with local mechanism interfaces creation and deletion methods
type Connect struct{}

// NewConnect - creates instance of local Connect
func NewConnect() *Connect {
	return &Connect{}
}

// CreateInterfaces - creates local interfaces pair
func (c *Connect) CreateInterfaces(srcName, dstName string) error {
	/* Create the VETH pair - host namespace */
	if err := netlink.LinkAdd(newVETH(srcName, dstName)); err != nil {
		return errors.Errorf("failed to create VETH pair - %v", err)
	}
	return nil
}

// DeleteInterfaces - deletes interfaces pair
func (c *Connect) DeleteInterfaces(ifaceName string) error {
	/* Get a link object for the interface */
	ifaceLink, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return errors.Errorf("failed to get link for %q - %v", ifaceName, err)
	}

	/* Delete the VETH pair - host namespace */
	if err := netlink.LinkDel(ifaceLink); err != nil {
		return errors.Errorf("local: failed to delete the VETH pair - %v", err)
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
