// Copyright 2018, 2019 VMware, Inc.
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

package common

import (
	"encoding/binary"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
)

func getEnv(key, description string, mandatory bool) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		if mandatory {
			logrus.Fatalf("Error getting %v: %v", key, ok)
		} else {
			logrus.Infof("%v not found.", key)
			return ""
		}
	}
	logrus.Infof("%s: %s", description, value)
	return value
}

// Ip2int converts and IP address to 32-bit unsignet integer
func Ip2int(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

// MechanismFromString creates a Mechanism from string
func MechanismFromString(mechanismName string) connection.MechanismType {
	mechanismName = strings.ToUpper(mechanismName) + "_INTERFACE"
	if mechanism, ok := connection.MechanismType_value[mechanismName]; ok {
		return connection.MechanismType(mechanism)
	}
	logrus.Infof("%s is not a valid MechanismType. Using Kernel Interface.", mechanismName)
	return connection.MechanismType_KERNEL_INTERFACE
}

// IsIPv6 function to check whether an IP is IPv6 or IPv4
func IsIPv6(address string) bool {
	return strings.Count(address, ":") >= 2
}
