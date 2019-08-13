//  Copyright (c) 2018 Cisco and/or its affiliates.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at:
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package vpp_l3

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ligato/vpp-agent/pkg/models"
)

// ModuleName is the module name used for models.
const ModuleName = "vpp"

var (
	ModelARPEntry = models.Register(&ARPEntry{}, models.Spec{
		Module:  ModuleName,
		Type:    "arp",
		Version: "v2",
	}, models.WithNameTemplate(
		"{{.Interface}}/{{.IpAddress}}",
	))

	ModelRoute = models.Register(&Route{}, models.Spec{
		Module:  ModuleName,
		Type:    "route",
		Version: "v2",
	}, models.WithNameTemplate(
		`vrf/{{.VrfId}}/dst/{{with ipnet .DstNetwork}}{{printf "%s/%d" .IP .MaskSize}}{{end}}/gw/{{.NextHopAddr}}`,
	))

	ModelProxyARP = models.Register(&ProxyARP{}, models.Spec{
		Module:  ModuleName,
		Type:    "proxyarp-global",
		Version: "v2",
	})

	ModelIPScanNeighbor = models.Register(&IPScanNeighbor{}, models.Spec{
		Module:  ModuleName,
		Type:    "ipscanneigh-global",
		Version: "v2",
	})

	ModelVrfTable = models.Register(&VrfTable{}, models.Spec{
		Module:  ModuleName,
		Type:    "vrf-table",
		Version: "v2",
	}, models.WithNameTemplate(
		`id/{{.Id}}/protocol/{{.Protocol}}`,
	))
)

// ProxyARPKey is key for global proxy arp
func ProxyARPKey() string {
	return models.Key(&ProxyARP{})
}

// ProxyARPKey is key for global ip scan neighbor
func IPScanNeighborKey() string {
	return models.Key(&IPScanNeighbor{})
}

// RouteKey returns the key used in ETCD to store vpp route for vpp instance.
func RouteKey(vrf uint32, dstNet string, nextHopAddr string) string {
	return models.Key(&Route{
		VrfId:       vrf,
		DstNetwork:  dstNet,
		NextHopAddr: nextHopAddr,
	})
}

// ArpEntryKey returns the key to store ARP entry
func ArpEntryKey(iface, ipAddr string) string {
	return models.Key(&ARPEntry{
		Interface: iface,
		IpAddress: ipAddr,
	})
}

// VrfTableKey returns the key used to represent configuration for VPP VRF table.
func VrfTableKey(id uint32, protocol VrfTable_Protocol) string {
	return models.Key(&VrfTable{
		Id:       id,
		Protocol: protocol,
	})
}

const (
	proxyARPInterfacePrefix   = "vpp/proxyarp/interface/"
	proxyARPInterfaceTemplate = proxyARPInterfacePrefix + "{iface}"
)

// ProxyARPInterfaceKey returns the key used to represent binding for interface with enabled proxy ARP.
func ProxyARPInterfaceKey(iface string) string {
	key := proxyARPInterfaceTemplate
	key = strings.Replace(key, "{iface}", iface, 1)
	return key
}

// ParseProxyARPInterfaceKey parses key representing binding for interface with enabled proxy ARP.
func ParseProxyARPInterfaceKey(key string) (iface string, isProxyARPInterfaceKey bool) {
	suffix := strings.TrimPrefix(key, proxyARPInterfacePrefix)
	if suffix != key && suffix != "" {
		return suffix, true
	}
	return "", false
}

// RouteVrfPrefix returns longest-common prefix of keys representing route that is written to given vrf table.
func RouteVrfPrefix(vrf uint32) string {
	return ModelRoute.KeyPrefix() + "vrf/" + fmt.Sprint(vrf) + "/"
}

// ParseRouteKey parses VRF label and route address from a route key.
func ParseRouteKey(key string) (vrfIndex string, dstNetAddr string, dstNetMask int, nextHopAddr string, isRouteKey bool) {
	if routeKey := strings.TrimPrefix(key, ModelRoute.KeyPrefix()); routeKey != key {
		keyParts := strings.Split(routeKey, "/")
		if len(keyParts) >= 7 &&
			keyParts[0] == "vrf" &&
			keyParts[2] == "dst" &&
			keyParts[5] == "gw" {
			if mask, err := strconv.Atoi(keyParts[4]); err == nil {
				return keyParts[1], keyParts[3], mask, keyParts[6], true
			}
		}
	}
	return "", "", 0, "", false
}
