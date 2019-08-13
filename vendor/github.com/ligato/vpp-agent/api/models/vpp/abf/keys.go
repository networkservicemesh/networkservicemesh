//  Copyright (c) 2019 Cisco and/or its affiliates.
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

package vpp_abf

import (
	"strconv"
	"strings"

	"github.com/ligato/vpp-agent/pkg/models"
)

// ModuleName is the name of the module used for models.
const ModuleName = "vpp.abfs"

var (
	ModelABF = models.Register(&ABF{}, models.Spec{
		Module:  ModuleName,
		Version: "v2",
		Type:    "abf",
	}, models.WithNameTemplate("{{.Index}}"))
)

// Key returns the prefix used in the ETCD to store VPP ACL-based forwarding
// config of a particular ABF in selected vpp instance.
func Key(index uint32) string {
	return models.Key(&ABF{
		Index: index,
	})
}

const (
	// ABF to interface template is a derived value key
	abfToInterfaceTemplate = "vpp/abf/{abf}/interface/{iface}"
)

const (
	// InvalidKeyPart is used in key for parts which are invalid
	InvalidKeyPart = "<invalid>"
)

// ToABFInterfaceKey returns key for ABF-to-interface
func ToInterfaceKey(abf uint32, iface string) string {
	if iface == "" {
		iface = InvalidKeyPart
	}
	key := abfToInterfaceTemplate
	key = strings.Replace(key, "{abf}", strconv.Itoa(int(abf)), 1)
	key = strings.Replace(key, "{iface}", iface, 1)
	return key
}

// ParseABFToInterfaceKey parses ABF-to-interface key
func ParseToInterfaceKey(key string) (abf, iface string, isABFToInterface bool) {
	parts := strings.Split(key, "/")
	if len(parts) >= 5 &&
		parts[0] == "vpp" && parts[1] == "abf" && parts[3] == "interface" {
		abf = parts[2]
		iface = strings.Join(parts[4:], "/")
		if iface != "" && abf != "" {
			return abf, iface, true
		}
	}
	return "", "", false
}
