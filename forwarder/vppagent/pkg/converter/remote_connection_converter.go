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
	"math"

	vpp_l3 "github.com/ligato/vpp-agent/api/models/vpp/l3"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/srv6"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	vpp_srv6 "github.com/ligato/vpp-agent/api/models/vpp/srv6"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
)

// RemoteConnectionConverter described the remote connection
type RemoteConnectionConverter struct {
	*connection.Connection
	name    string
	tapName string
	side    ConnectionContextSide
}

// NewRemoteConnectionConverter creates a new remote connection coverter
func NewRemoteConnectionConverter(c *connection.Connection, name, tapName string, side ConnectionContextSide) *RemoteConnectionConverter {
	return &RemoteConnectionConverter{
		Connection: c,
		name:       name,
		tapName:    tapName,
		side:       side,
	}
}

func (c *RemoteConnectionConverter) checkMechanism() bool {
	mechanisms := []string{vxlan.MECHANISM, srv6.MECHANISM}
	for _, m := range mechanisms {
		if m == c.GetMechanism().GetType() {
			return true
		}
	}
	return false
}

// ToDataRequest handles the data request
func (c *RemoteConnectionConverter) ToDataRequest(rv *configurator.Config, connect bool) (*configurator.Config, error) {
	if c == nil {
		return rv, errors.New("RemoteConnectionConverter cannot be nil")
	}
	if err := c.IsComplete(); err != nil {
		return rv, err
	}

	if !c.checkMechanism() {
		return rv, errors.Errorf("attempt to use not supported Connection.Mechanism.Type %s", c.GetMechanism().GetType())
	}

	if rv == nil {
		rv = &configurator.Config{}
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}

	switch c.GetMechanism().GetType() {
	case vxlan.MECHANISM:
		m := vxlan.ToMechanism(c.GetMechanism())
		// If the remote Connection is DESTINATION Side then srcip/dstip match the Connection
		srcip, _ := m.SrcIP()
		dstip, _ := m.DstIP()
		if c.side == SOURCE {
			// If the remote Connection is DESTINATION Side then srcip/dstip need to be flipped from the Connection
			srcip, _ = m.DstIP()
			dstip, _ = m.SrcIP()
		}
		vni, _ := m.VNI()

		logrus.Infof("m.GetParameters()[%s]: %s", vxlan.SrcIP, srcip)
		logrus.Infof("m.GetParameters()[%s]: %s", vxlan.DstIP, dstip)
		logrus.Infof("m.GetParameters()[%s]: %d", vxlan.VNI, vni)

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
	case srv6.MECHANISM:
		m := srv6.ToMechanism(c.GetMechanism())

		dstHostLocalSID, _ := m.DstHostLocalSID()
		hardwareAddress, _ := m.DstHardwareAddress()
		srcBSID, _ := m.SrcBSID()
		srcLocalSID, _ := m.SrcLocalSID()
		dstLocalSID, _ := m.DstLocalSID()

		if c.side == SOURCE {
			// If the remote Connection is DESTINATION Side then src/dst addresses need to be flipped from the Connection
			dstHostLocalSID, _ = m.SrcHostLocalSID()
			hardwareAddress, _ = m.SrcHardwareAddress()
			srcBSID, _ = m.DstBSID()
			srcLocalSID, _ = m.DstLocalSID()
			dstLocalSID, _ = m.SrcLocalSID()
		}

		logrus.Infof("m.GetParameters()[%s]: %s", srv6.DstHostLocalSID, dstHostLocalSID)
		logrus.Infof("m.GetParameters()[%s]: %s", srv6.DstHardwareAddress, hardwareAddress)
		logrus.Infof("m.GetParameters()[%s]: %s", srv6.SrcBSID, srcBSID)
		logrus.Infof("m.GetParameters()[%s]: %s", srv6.SrcLocalSID, srcLocalSID)
		logrus.Infof("m.GetParameters()[%s]: %s", srv6.DstLocalSID, dstLocalSID)

		rv.VppConfig.Srv6Localsids = []*vpp_srv6.LocalSID{
			{
				Sid: srcLocalSID,
				EndFunction: &vpp_srv6.LocalSID_EndFunction_DX2{
					EndFunction_DX2: &vpp_srv6.LocalSID_EndDX2{
						VlanTag:           math.MaxUint32,
						OutgoingInterface: c.tapName,
					},
				},
			},
		}
		rv.VppConfig.Srv6Policies = []*vpp_srv6.Policy{
			{
				Bsid: srcBSID,
				SegmentLists: []*vpp_srv6.Policy_SegmentList{
					{
						Segments: []string{
							dstHostLocalSID,
							dstLocalSID,
						},
						Weight: 0,
					},
				},
				SrhEncapsulation: true,
			},
		}

		rv.VppConfig.Srv6Steerings = []*vpp_srv6.Steering{
			{
				Name: c.name,
				PolicyRef: &vpp_srv6.Steering_PolicyBsid{
					PolicyBsid: srcBSID,
				},
				Traffic: &vpp_srv6.Steering_L2Traffic_{
					L2Traffic: &vpp_srv6.Steering_L2Traffic{
						InterfaceName: c.tapName,
					},
				},
			},
		}

		if connect {
			rv.VppConfig.Vrfs = []*vpp_l3.VrfTable{
				{
					Id:       math.MaxUint32,
					Protocol: vpp_l3.VrfTable_IPV6,
					Label:    "SRv6 steering of IP6 prefixes through BSIDs",
				},
			}

			rv.VppConfig.Routes = append(rv.VppConfig.Routes, &vpp.Route{
				Type:              vpp_l3.Route_INTER_VRF,
				OutgoingInterface: "mgmt",
				DstNetwork:        dstHostLocalSID + "/128",
				Weight:            1,
				NextHopAddr:       dstHostLocalSID,
			})

			rv.VppConfig.Arps = append(rv.VppConfig.Arps, &vpp.ARPEntry{
				Interface:   "mgmt",
				IpAddress:   dstHostLocalSID,
				PhysAddress: hardwareAddress,
				Static:      true,
			})
		}
	}

	return rv, nil
}
