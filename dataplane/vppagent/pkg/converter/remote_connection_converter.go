// Copyright 2018 VMware, Inc.
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

package converter

import (
	fmt "fmt"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"github.com/sirupsen/logrus"
)

// RemoteConnectionConverter descibed the remote connection
type RemoteConnectionConverter struct {
	*connection.Connection
	name string
}

// NewRemoteConnectionConverter creates a new remote connection coverter
func NewRemoteConnectionConverter(c *connection.Connection, name string) *RemoteConnectionConverter {
	return &RemoteConnectionConverter{
		Connection: c,
		name:       name,
	}
}

// ToDataRequest handles the data request
func (c *RemoteConnectionConverter) ToDataRequest(rv *rpc.DataRequest) (*rpc.DataRequest, error) {
	if c == nil {
		return rv, fmt.Errorf("RemoteConnectionConverter cannot be nil")
	}
	if err := c.IsComplete(); err != nil {
		return rv, err
	}
	if c.GetMechanism().GetType() != connection.MechanismType_VXLAN {
		return rv, fmt.Errorf("RemoteConnectionConverter supports only VXLAN. Attempt to use Connection.Mechanism.Type %s", c.GetMechanism().GetType())
	}
	if rv == nil {
		rv = &rpc.DataRequest{}
	}

	m := c.GetMechanism()

	srcip, _ := m.SrcIP()
	dstip, _ := m.DstIP()
	vni, _ := m.VNI()

	logrus.Infof("m.GetParameters()[%s]: %s", connection.VXLANSrcIP, srcip)
	logrus.Infof("m.GetParameters()[%s]: %s", connection.VXLANDstIP, dstip)
	logrus.Infof("m.GetParameters()[%s]: %d", connection.VXLANVNI, vni)

	rv.Interfaces = append(rv.Interfaces, &interfaces.Interfaces_Interface{
		Name:    c.name,
		Type:    interfaces.InterfaceType_VXLAN_TUNNEL,
		Enabled: true,
		Vxlan: &interfaces.Interfaces_Interface_Vxlan{
			SrcAddress: srcip,
			DstAddress: dstip,
			Vni:        vni,
		},
	})

	return rv, nil
}
