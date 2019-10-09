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
	"strings"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	advertiseNseNameEnv   = "ADVERTISE_NSE_NAME"
	advertiseNseLabelsEnv = "ADVERTISE_NSE_LABELS"
	outgoingNscNameEnv    = "OUTGOING_NSC_NAME"
	outgoingNscLabelsEnv  = "OUTGOING_NSC_LABELS"
	nscInterfaceName      = "NSC_INTERFACE_NAME"
	mechanismTypeEnv      = "MECHANISM_TYPE"
	ipAddressEnv          = "IP_ADDRESS"
	routesEnv             = "ROUTES"
)

// NSConfiguration contains the full configuration used in the SDK
type NSConfiguration struct {
	NsmServerSocket    string
	NsmClientSocket    string
	Workspace          string
	AdvertiseNseName   string
	OutgoingNscName    string
	AdvertiseNseLabels string
	OutgoingNscLabels  string
	NscInterfaceName   string
	MechanismType      string
	IPAddress          string
	Routes             []string
}

// FromEnv creates a new NSConfiguration and fills all unset options from the env variables
func FromEnv() *NSConfiguration {
	return (&NSConfiguration{}).FromEnv()
}

// FromEnv fills all unset options from the env variables
func (configuration *NSConfiguration) FromEnv() *NSConfiguration {
	if configuration == nil {
		return nil
	}

	if configuration.NsmServerSocket == "" {
		configuration.NsmServerSocket = getEnv(NsmServerSocketEnv, "nsmServerSocket", true)
	}

	if configuration.NsmClientSocket == "" {
		configuration.NsmClientSocket = getEnv(NsmClientSocketEnv, "nsmClientSocket", true)
	}

	if configuration.Workspace == "" {
		configuration.Workspace = getEnv(WorkspaceEnv, "workspace", true)
	}

	if configuration.AdvertiseNseName == "" {
		configuration.AdvertiseNseName = getEnv(advertiseNseNameEnv, "Advertise Network Service Name", false)
	}

	if configuration.OutgoingNscName == "" {
		configuration.OutgoingNscName = getEnv(outgoingNscNameEnv, "Outgoing Network Service Name", false)
	}

	if configuration.AdvertiseNseLabels == "" {
		configuration.AdvertiseNseLabels = getEnv(advertiseNseLabelsEnv, "Advertise labels", false)
	}

	if configuration.OutgoingNscLabels == "" {
		configuration.OutgoingNscLabels = getEnv(outgoingNscLabelsEnv, "Outgoing labels", false)
	}

	if configuration.NscInterfaceName == "" {
		configuration.NscInterfaceName = getEnv(nscInterfaceName, "NSC Interface name", false)
	}

	if configuration.MechanismType == "" {
		configuration.MechanismType = getEnv(mechanismTypeEnv, "Outgoing mechanism type", false)
	}

	if len(configuration.IPAddress) == 0 {
		configuration.IPAddress = getEnv(ipAddressEnv, "IP Address", false)
	}

	if len(configuration.Routes) == 0 {
		raw := getEnv(routesEnv, "Routes", false)
		if len(raw) > 1 {
			configuration.Routes = strings.Split(raw, ",")
		}
	}
	return configuration
}

func FromNSUrl(url *tools.NSUrl) *NSConfiguration {
	return (&NSConfiguration{}).FromNSUrl(url)
}

func (configuration *NSConfiguration) FromNSUrl(url *tools.NSUrl) *NSConfiguration {
	if configuration == nil {
		return nil
	}
	configuration.OutgoingNscName = url.NsName
	configuration.NscInterfaceName = url.Intf
	var labels strings.Builder
	separator := false
	for k, v := range url.Params {
		if separator {
			labels.WriteRune(',')
		} else {
			separator = true
		}
		labels.WriteString(k)
		labels.WriteRune('=')
		labels.WriteString(v[0])
	}
	configuration.OutgoingNscLabels = labels.String()
	return configuration
}
