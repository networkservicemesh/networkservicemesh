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
	"fmt"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
)

// SupportedMechanisms by Dataplane (add new mechanisms next way "connection.MechanismType_VXLAN | connection.MechanismType_SRV6 | ...")
const SupportedMechanisms = connection.MechanismType_VXLAN

// RemoteConnectionConverter described the remote connection
type RemoteConnectionConverter struct {
	*connection.Connection
	name string
	side ConnectionContextSide
}

// NewRemoteConnectionConverter creates a new remote connection coverter
func NewRemoteConnectionConverter(c *connection.Connection, name string, side ConnectionContextSide) *RemoteConnectionConverter {
	return &RemoteConnectionConverter{
		Connection: c,
		name:       name,
		side:       side,
	}
}

// ToDataRequest handles the data request
func (c *RemoteConnectionConverter) ToDataRequest(rv *configurator.Config, connect bool) (*configurator.Config, error) {
	if c == nil {
		return rv, fmt.Errorf("RemoteConnectionConverter cannot be nil")
	}
	if err := c.IsComplete(); err != nil {
		return rv, err
	}
	if c.GetMechanism().GetType()&SupportedMechanisms == 0 {
		return rv, fmt.Errorf("attempt to use not supported Connection.Mechanism.Type %s", c.GetMechanism().GetType())
	}
	if rv == nil {
		rv = &configurator.Config{}
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}

	m := c.GetMechanism()

	srcip, dstip, vni, err := getParameters(m, c.side)
	if err != nil {
		return rv, nil
	}

	logrus.Infof("m.GetParameters()[%s]: %s", connection.VXLANSrcIP, srcip)
	logrus.Infof("m.GetParameters()[%s]: %s", connection.VXLANDstIP, dstip)
	logrus.Infof("m.GetParameters()[%s]: %d", connection.VXLANVNI, vni)

	rv.VppConfig.Interfaces = append(rv.VppConfig.Interfaces, &vpp.Interface{
		Name:    c.name,
		Type:    vpp_interfaces.Interface_VXLAN_TUNNEL,
		Enabled: true,
		Link: &vpp_interfaces.Interface_Vxlan{
			Vxlan: &vpp_interfaces.VxlanLink{
				SrcAddress: srcip,
				DstAddress: dstip,
				Vni:        vni,
			},
		},
	})

	return rv, nil
}

func getParameters(m *connection.Mechanism, side ConnectionContextSide) (string, string, uint32, error) {
	var srcip, dstip string
	var useExtIP bool
	var vni uint32
	var err error

	useExtIP, err = m.UseExtIP()
	if err != nil {
		return srcip, dstip, vni, err
	}

	srcip, err = m.SrcIP()
	if err != nil {
		return srcip, dstip, vni, err
	}
	dstip, err = m.DstIP()
	if err != nil {
		return srcip, dstip, vni, err
	}

	if useExtIP {
		extip, err1 := m.DstExtIP()
		if err1 == nil {
			dstip = extip
		}
	}

	if side == SOURCE {
		srcip, err = m.DstIP()
		if err != nil {
			return srcip, dstip, vni, err
		}
		dstip, err = m.SrcIP()
		if err != nil {
			return srcip, dstip, vni, err
		}

		if useExtIP {
			extip, err1 := m.SrcExtIP()
			if err1 == nil {
				dstip = extip
			}
		}
	}

	vni, err = m.VNI()
	if err != nil {
		return srcip, dstip, vni, err
	}

	return srcip, dstip, vni, err
}
