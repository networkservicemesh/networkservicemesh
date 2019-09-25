// Copyright 2018-2019 VMware, Inc.
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

package connection

import (
	"fmt"
	"net"
	"strconv"

	"github.com/gogo/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm/connection"
)

// IsRemote returns if mechanism type is remote
func (t MechanismType) IsRemote() bool {
	return true
}

// IsRemote returns if mechanism is remote
func (m *Mechanism) IsRemote() bool {
	return true
}

// Equals returns if mechanism equals given mechanism
func (m *Mechanism) Equals(mechanism connection.Mechanism) bool {
	if other, ok := mechanism.(*Mechanism); ok {
		return proto.Equal(m, other)
	}

	return false
}

// Clone clones mechanism
func (m *Mechanism) Clone() connection.Mechanism {
	return proto.Clone(m).(*Mechanism)
}

// GetMechanismType returns mechanism type
func (m *Mechanism) GetMechanismType() connection.MechanismType {
	return m.Type
}

// SetMechanismType sets mechanism type
func (m *Mechanism) SetMechanismType(mechanismType connection.MechanismType) {
	m.Type = mechanismType.(MechanismType)
}

// SetParameters sets mechanism parameters
func (m *Mechanism) SetParameters(parameters map[string]string) {
	m.Parameters = parameters
}

// IsValid checks if mechanism is valid
func (m *Mechanism) IsValid() error {
	if m == nil {
		return fmt.Errorf("mechanism cannot be nil")
	}
	if m.GetParameters() == nil {
		return fmt.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	switch m.GetType() {
	case MechanismType_VXLAN:
		if _, err := m.SrcIP(); err != nil {
			return fmt.Errorf("mechanism.Type %s requires mechanism.Parameters[%s] for VXLAN tunnel, caused by: %+v", m.GetType(), VXLANSrcIP, err)
		}

		if _, err := m.DstIP(); err != nil {
			return fmt.Errorf("mechanism.Type %s requires mechanism.Parameters[%s] for VXLAN tunnel, caused by: %+v", m.GetType(), VXLANDstIP, err)
		}

		if _, err := m.VNI(); err != nil {
			return fmt.Errorf("mechanism.Type %s requires mechanism.Parameters[%s] for VXLAN tunnel, caused by: %+v", m.GetType(), VXLANVNI, err)
		}
	}

	return nil
}

// SrcIP returns the source IP parameter of the Mechanism
func (m *Mechanism) SrcIP() (string, error) {
	return m.getIPParameter(VXLANSrcIP)
}

// DstIP returns the destination IP parameter of the Mechanism
func (m *Mechanism) DstIP() (string, error) {
	return m.getIPParameter(VXLANDstIP)
}

func (m *Mechanism) getIPParameter(name string) (string, error) {
	if m == nil {
		return "", fmt.Errorf("mechanism cannot be nil")
	}

	if m.GetParameters() == nil {
		return "", fmt.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	ip, ok := m.Parameters[name]
	if !ok {
		return "", fmt.Errorf("mechanism.Type %s requires mechanism.Parameters[%s] for the VXLAN tunnel", m.GetType(), name)
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("mechanism.Parameters[%s] must be a valid IPv4 or IPv6 address, instead was: %s: %v", name, ip, m)
	}

	return ip, nil
}

// VNI returns the VNI parameter of the Mechanism
func (m *Mechanism) VNI() (uint32, error) {
	if m == nil {
		return 0, fmt.Errorf("mechanism cannot be nil")
	}

	if m.GetParameters() == nil {
		return 0, fmt.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	vxlanvni, ok := m.Parameters[VXLANVNI]
	if !ok {
		return 0, fmt.Errorf("mechanism.Type %s requires mechanism.Parameters[%s]", m.GetType(), VXLANVNI)
	}

	vni, err := strconv.ParseUint(vxlanvni, 10, 24)

	if err != nil {
		return 0, fmt.Errorf("mechanism.Parameters[%s] must be a valid 24-bit unsigned integer, instead was: %s: %v", VXLANVNI, vxlanvni, m)
	}

	return uint32(vni), nil
}
