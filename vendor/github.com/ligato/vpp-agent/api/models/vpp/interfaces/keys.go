// Copyright (c) 2017 Cisco and/or its affiliates.
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

package vpp_interfaces

import (
	"net"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/jsonpb"

	"github.com/ligato/vpp-agent/pkg/models"
)

// ModuleName is the module name used for models.
const ModuleName = "vpp"

var (
	ModelInterface = models.Register(&Interface{}, models.Spec{
		Module:  ModuleName,
		Version: "v2",
		Type:    "interfaces",
	})
)

// InterfaceKey returns the key used in NB DB to store the configuration of the
// given vpp interface.
func InterfaceKey(name string) string {
	return models.Key(&Interface{
		Name: name,
	})
}

/* Interface State */
const (
	// StatePrefix is a key prefix used in NB DB to store interface states.
	StatePrefix = "vpp/status/v2/interface/"
)

/* Interface Error */
const (
	// ErrorPrefix is a key prefix used in NB DB to store interface errors.
	ErrorPrefix = "vpp/status/v2/interface/error/"
)

/* Interface Address (derived) */
const (
	// addressKeyTemplate is a template for (derived) key representing assigned
	// IP addresses to an interface.
	addressKeyTemplate = "vpp/interface/{iface}/address/{address}"
)

/* Interface VRF (derived) */
const (
	// vrfTKeyTemplatePrefix is a prefix of the template used to construct
	// key representing assignment of an interface into a VRF table.
	// The prefix includes the source interface name, but not details about the
	// target VRF.
	vrfTKeyTemplatePrefix = "vpp/interface/{iface}/vrf/"

	// vrfKeyTemplate is a template for (derived) key representing assignment
	// of a VPP interface into a VRF table.
	vrfKeyTemplate = vrfTKeyTemplatePrefix + "{vrf}/ip-version/{ip-version}"

	// inheritedVrfKeyTemplate is a template for (derived) key representing
	// assignment of an (unnumbered) VPP interface into a VRF table, ID of which
	// is given by the configuration of a referenced (numbered) interface.
	inheritedVrfKeyTemplate = vrfTKeyTemplatePrefix + "from-interface/{from-iface}"

	vrfIPv4 = "v4"
	vrfIPv6 = "v6"
	vrfBoth = "both"
)

/* Unnumbered interface (derived) */
const (
	// UnnumberedKeyPrefix is used as a common prefix for keys derived from
	// interfaces to represent unnumbered interfaces.
	UnnumberedKeyPrefix = "vpp/interface/unnumbered/"
)

/* Bond interface enslavement (derived) */
const (
	// BondedInterfacePrefix is used as a common prefix for keys derived from
	// interfaces to represent interface slaves for bond interface.
	BondedInterfacePrefix = "vpp/bond/{bond}/interface/{iface}/"
)

/* DHCP (client - derived, lease - notification) */
const (
	// DHCPClientKeyPrefix is used as a common prefix for keys derived from
	// interfaces to represent enabled DHCP clients.
	DHCPClientKeyPrefix = "vpp/interface/dhcp-client/"

	// DHCPLeaseKeyPrefix is used as a common prefix for keys representing
	// notifications with DHCP leases.
	DHCPLeaseKeyPrefix = "vpp/interface/dhcp-lease/"
)

const (
	// InvalidKeyPart is used in key for parts which are invalid
	InvalidKeyPart = "<invalid>"
)

/* Interface Error */

// InterfaceErrorKey returns the key used in NB DB to store the interface errors.
func InterfaceErrorKey(iface string) string {
	if iface == "" {
		iface = InvalidKeyPart
	}
	return ErrorPrefix + iface
}

/* Interface State */

// InterfaceStateKey returns the key used in NB DB to store the state data of the
// given vpp interface.
func InterfaceStateKey(iface string) string {
	if iface == "" {
		iface = InvalidKeyPart
	}
	return StatePrefix + iface
}

/* Interface Address (derived) */

// InterfaceAddressPrefix returns longest-common prefix of keys representing
// assigned IP addresses to a specific VPP interface.
func InterfaceAddressPrefix(iface string) string {
	return InterfaceAddressKey(iface, "")
}

// InterfaceAddressKey returns key representing IP address assigned to VPP interface.
func InterfaceAddressKey(iface string, address string) string {
	if iface == "" {
		iface = InvalidKeyPart
	}

	// construct key without validating the IP address
	key := strings.Replace(addressKeyTemplate, "{iface}", iface, 1)
	key = strings.Replace(key, "{address}", address, 1)
	return key
}

// ParseInterfaceAddressKey parses interface address from key derived
// from interface by InterfaceAddressKey().
func ParseInterfaceAddressKey(key string) (iface string, ipAddr net.IP, ipAddrNet *net.IPNet, invalidIP, isAddrKey bool) {
	parts := strings.Split(key, "/")
	if len(parts) < 4 || parts[0] != "vpp" || parts[1] != "interface" {
		return
	}

	addrIdx := -1
	for idx, part := range parts {
		if part == "address" {
			addrIdx = idx
			break
		}
	}
	if addrIdx == -1 {
		return
	}
	isAddrKey = true

	// parse interface name
	iface = strings.Join(parts[2:addrIdx], "/")
	if iface == "" {
		iface = InvalidKeyPart
	}

	// parse IP address
	var err error
	ipAddr, ipAddrNet, err = net.ParseCIDR(strings.Join(parts[addrIdx+1:], "/"))
	if err != nil {
		invalidIP = true
	}

	return
}

/* Interface VRF (derived) */

// InterfaceVrfKeyPrefix returns prefix of the key representing assignment
// of the given interface into unspecified VRF table.
func InterfaceVrfKeyPrefix(iface string) string {
	return strings.Replace(vrfTKeyTemplatePrefix, "{iface}", iface, 1)
}

// InterfaceVrfKey returns key representing assignment of the given interface
// into the given VRF.
func InterfaceVrfKey(iface string, vrf int, ipv4, ipv6 bool) string {
	if iface == "" {
		iface = InvalidKeyPart
	}
	ipVer := InvalidKeyPart
	if ipv4 && ipv6 {
		ipVer = vrfBoth
	} else if ipv4 {
		ipVer = vrfIPv4
	} else if ipv6 {
		ipVer = vrfIPv6
	}

	vrfTableID := InvalidKeyPart
	if vrf >= 0 {
		vrfTableID = strconv.Itoa(vrf)
	}

	key := strings.Replace(vrfKeyTemplate, "{iface}", iface, 1)
	key = strings.Replace(key, "{vrf}", vrfTableID, 1)
	key = strings.Replace(key, "{ip-version}", ipVer, 1)
	return key
}

// ParseInterfaceVrfKey parses details from key derived from interface by
// InterfaceVrfKey().
func ParseInterfaceVrfKey(key string) (iface string, vrf int, ipv4, ipv6, isIfaceVrfKey bool) {
	parts := strings.Split(key, "/")
	if len(parts) < 7 || parts[0] != "vpp" || parts[1] != "interface" {
		return
	}

	vrfIdx := -1
	ipVerIdx := -1
	for idx, part := range parts {
		if part == "vrf" {
			vrfIdx = idx
		}
		if part == "ip-version" {
			ipVerIdx = idx
		}
	}
	if vrfIdx == -1 || ipVerIdx != len(parts)-2 {
		return
	}
	isIfaceVrfKey = true

	// parse interface name
	iface = strings.Join(parts[2:vrfIdx], "/")
	if iface == "" {
		iface = InvalidKeyPart
	}

	// parse VRF table ID
	var err error
	vrf, err = strconv.Atoi(parts[vrfIdx+1])
	if err != nil {
		vrf = -1
	}

	// parse IP version
	switch parts[ipVerIdx+1] {
	case vrfBoth:
		ipv4 = true
		ipv6 = true
	case vrfIPv4:
		ipv4 = true
	case vrfIPv6:
		ipv6 = true
	}
	return
}

// InterfaceInheritedVrfKey returns key representing assignment of the given
// interface into a VRF inherited from another interface.
// Used by unnumbered interfaces.
func InterfaceInheritedVrfKey(iface string, fromIface string) string {
	if iface == "" {
		iface = InvalidKeyPart
	}
	if fromIface == "" {
		fromIface = InvalidKeyPart
	}
	key := strings.Replace(inheritedVrfKeyTemplate, "{iface}", iface, 1)
	key = strings.Replace(key, "{from-iface}", fromIface, 1)
	return key
}

// ParseInterfaceInheritedVrfKey parses details from key derived from interface by
// InterfaceInheritedVrfKey().
func ParseInterfaceInheritedVrfKey(key string) (iface, fromIface string, isIfaceInherVrfKey bool) {
	parts := strings.Split(key, "/")
	if len(parts) < 6 || parts[0] != "vpp" || parts[1] != "interface" {
		return
	}

	vrfIdx := -1
	for idx, part := range parts {
		if part == "vrf" {
			vrfIdx = idx
			break
		}
	}
	if vrfIdx == -1 || vrfIdx+2 >= len(parts) || parts[vrfIdx+1] != "from-interface" {
		return
	}
	isIfaceInherVrfKey = true

	// parse interface name
	iface = strings.Join(parts[2:vrfIdx], "/")
	if iface == "" {
		iface = InvalidKeyPart
	}

	// parse from-interface name
	fromIface = strings.Join(parts[vrfIdx+2:], "/")
	if fromIface == "" {
		fromIface = InvalidKeyPart
	}
	return
}

/* Unnumbered interface (derived) */

// UnnumberedKey returns key representing unnumbered interface.
func UnnumberedKey(iface string) string {
	if iface == "" {
		iface = InvalidKeyPart
	}
	return UnnumberedKeyPrefix + iface
}

// ParseNameFromUnnumberedKey returns suffix of the key.
func ParseNameFromUnnumberedKey(key string) (iface string, isUnnumberedKey bool) {
	suffix := strings.TrimPrefix(key, UnnumberedKeyPrefix)
	if suffix != key && suffix != "" {
		return suffix, true
	}
	return
}

/* Bond slave interface (derived) */

// BondedInterfaceKey returns a key with bond and slave interface set
func BondedInterfaceKey(bondIf, slaveIf string) string {
	if bondIf == "" {
		bondIf = InvalidKeyPart
	}
	if slaveIf == "" {
		slaveIf = InvalidKeyPart
	}
	key := strings.Replace(BondedInterfacePrefix, "{bond}", bondIf, 1)
	key = strings.Replace(key, "{iface}", slaveIf, 1)
	return key
}

// ParseBondedInterfaceKey returns names of interfaces of the key.
func ParseBondedInterfaceKey(key string) (bondIf, slaveIf string, isBondSlaveInterfaceKey bool) {
	keyComps := strings.Split(key, "/")
	if len(keyComps) >= 5 && keyComps[0] == "vpp" && keyComps[1] == "bond" && keyComps[3] == "interface" {
		slaveIf = strings.Join(keyComps[4:], "/")
		return keyComps[2], slaveIf, true
	}
	return "", "", false
}

/* DHCP (client - derived, lease - notification) */

// DHCPClientKey returns a (derived) key used to represent enabled DHCP lease.
func DHCPClientKey(iface string) string {
	if iface == "" {
		iface = InvalidKeyPart
	}
	return DHCPClientKeyPrefix + iface
}

// ParseNameFromDHCPClientKey returns suffix of the key.
func ParseNameFromDHCPClientKey(key string) (iface string, isDHCPClientKey bool) {
	if suffix := strings.TrimPrefix(key, DHCPClientKeyPrefix); suffix != key && suffix != "" {
		return suffix, true
	}
	return
}

// DHCPLeaseKey returns a key used to represent DHCP lease for the given interface.
func DHCPLeaseKey(iface string) string {
	if iface == "" {
		iface = InvalidKeyPart
	}
	return DHCPLeaseKeyPrefix + iface
}

// ParseNameFromDHCPLeaseKey returns suffix of the key.
func ParseNameFromDHCPLeaseKey(key string) (iface string, isDHCPLeaseKey bool) {
	if suffix := strings.TrimPrefix(key, DHCPLeaseKeyPrefix); suffix != key && suffix != "" {
		return suffix, true
	}
	return
}

// MarshalJSON ensures that field of type 'oneOf' is correctly marshaled
// by using gogo lib marshaller
func (m *Interface) MarshalJSON() ([]byte, error) {
	marshaller := &jsonpb.Marshaler{}
	str, err := marshaller.MarshalToString(m)
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}

// UnmarshalJSON ensures that field of type 'oneOf' is correctly unmarshaled
func (m *Interface) UnmarshalJSON(data []byte) error {
	return jsonpb.UnmarshalString(string(data), m)
}
