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

package connection

import (
	fmt "fmt"
	"net"
	"strconv"
)

// IsValid - Did you tell me enough that some other party in the chain can fill in the blanks
func (c *Connection) IsValid() error {
	if c == nil {
		return fmt.Errorf("Connection cannot be nil")
	}
	if c.GetNetworkService() == "" {
		return fmt.Errorf("Connection.NetworkService cannot be empty: %v", c)
	}

	if c.GetMechanism() != nil {
		if err := c.GetMechanism().isValid(); err != nil {
			return fmt.Errorf("Invalid Mechanism in %v: %s", c, err)
		}
	}
	return nil
}

// IsComplete - Have I been told enough to actually give you what you asked for
func (c *Connection) IsComplete() error {
	if err := c.IsValid(); err != nil {
		return err
	}

	if c.GetId() == "" {
		return fmt.Errorf("Connection.Id cannot be empty: %v", c)
	}

	if c.GetContext() == nil {
		return fmt.Errorf("Connection.Context cannot be nil: %v", c)
	}

	return nil
}

func (m *Mechanism) isValid() error {
	if m == nil {
		return fmt.Errorf("Mechanism cannot be nil")
	}
	if m.GetParameters() == nil {
		return fmt.Errorf("Mechanism.Parameters cannot be nil: %v", m)
	}

	if m.GetType() == MechanismType_VXLAN {
		if _, err := m.SrcIP(); err != nil {
			return fmt.Errorf("Mechanism.Type %s requires Mechanism.Parameters[%s] for VXLAN tunnel, caused by: %+v", m.GetType(), VXLANSrcIP, err)
		}

		if _, err := m.DstIP(); err != nil {
			return fmt.Errorf("Mechanism.Type %s requires Mechanism.Parameters[%s] for VXLAN tunnel, caused by: %+v", m.GetType(), VXLANDstIP, err)
		}

		if _, err := m.VNI(); err != nil {
			return fmt.Errorf("Mechanism.Type %s requires Mechanism.Parameters[%s] for VXLAN tunnel, caused by: %+v", m.GetType(), VXLANVNI, err)
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
		return "", fmt.Errorf("Mechanism cannot be nil")
	}
	if m.GetParameters() == nil {
		return "", fmt.Errorf("Mechanism.Parameters cannot be nil: %v", m)
	}

	ip, ok := m.Parameters[name]
	if !ok {
		return "", fmt.Errorf("Mechanism.Type %s requires Mechanism.Parameters[%s] for the VXLAN tunnel", m.GetType(), name)
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("Mechanism.Parameters[%s] must be a valid IPv4 or IPv6 address, instead was: %s: %v", name, ip, m)
	}

	return ip, nil
}

// VNI returns the VNI parameter of the Mechanism
func (m *Mechanism) VNI() (uint32, error) {
	if m == nil {
		return 0, fmt.Errorf("Mechanism cannot be nil")
	}
	if m.GetParameters() == nil {
		return 0, fmt.Errorf("Mechanism.Parameters cannot be nil: %v", m)
	}

	vxlanvni, ok := m.Parameters[VXLANVNI]
	if !ok {
		return 0, fmt.Errorf("Mechanism.Type %s requires Mechanism.Parameters[%s]", m.GetType(), VXLANVNI)
	}

	vni, err := strconv.ParseUint(vxlanvni, 10, 24)

	if err != nil {
		return 0, fmt.Errorf("Mechanism.Parameters[%s] must be a valid 24-bit unsigned integer, instead was: %s: %v", VXLANVNI, vxlanvni, m)
	}

	return uint32(vni), nil
}
