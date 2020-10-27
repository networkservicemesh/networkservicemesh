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
	namespaceEnv              = "NSM_NAMESPACE"
	endpointNetworkServiceEnv = "ENDPOINT_NETWORK_SERVICE"
	endpointLabelsEnv         = "ENDPOINT_LABELS"
	clientNetworkServiceEnv   = "CLIENT_NETWORK_SERVICE"
	clientLabelsEnv           = "CLIENT_LABELS"
	nscInterfaceNameEnv       = "NSC_INTERFACE_NAME"
	mechanismTypeEnv          = "MECHANISM_TYPE"
	ipAddressEnv              = "IP_ADDRESS"
	routesEnv                 = "ROUTES"
	podNameEnv                = "POD_NAME"
	memifModeEnv              = "MEMIF_MODE"
)

// NSConfiguration contains the full configuration used in the SDK
type NSConfiguration struct {
	NsmServerSocket        string
	NsmClientSocket        string
	Workspace              string
	EndpointNetworkService string
	ClientNetworkService   string
	EndpointLabels         string
	ClientLabels           string
	NscInterfaceName       string
	MechanismType          string
	IPAddress              string
	Routes                 []string
	PodName                string
	Namespace              string
	MemifMode              string
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

	if configuration.EndpointNetworkService == "" {
		configuration.EndpointNetworkService = getEnv(endpointNetworkServiceEnv, "Advertise Network Service Name", false)
	}

	if configuration.ClientNetworkService == "" {
		configuration.ClientNetworkService = getEnv(clientNetworkServiceEnv, "Outgoing Network Service Name", false)
	}

	if configuration.EndpointLabels == "" {
		configuration.EndpointLabels = getEnv(endpointLabelsEnv, "Advertise labels", false)
	}

	if configuration.ClientLabels == "" {
		configuration.ClientLabels = getEnv(clientLabelsEnv, "Outgoing labels", false)
	}

	if configuration.NscInterfaceName == "" {
		configuration.NscInterfaceName = getEnv(nscInterfaceNameEnv, "NSC Interface name", false)
	}

	if configuration.MechanismType == "" {
		configuration.MechanismType = getEnv(mechanismTypeEnv, "Outgoing mechanism type", false)
	}

	if len(configuration.IPAddress) == 0 {
		configuration.IPAddress = getEnv(ipAddressEnv, "IP Address", false)
	}

	if configuration.PodName == "" {
		configuration.PodName = getEnv(podNameEnv, "Pod name", false)
	}

	if configuration.Namespace == "" {
		configuration.Namespace = getEnv(namespaceEnv, "Namespace", false)
	}

	if configuration.MemifMode == "" {
		configuration.MemifMode = getEnv(memifModeEnv, "Memif mode (L2/L3)", false)
	}

	if len(configuration.Routes) == 0 {
		raw := getEnv(routesEnv, "Routes", false)
		if len(raw) > 1 {
			configuration.Routes = strings.Split(raw, ",")
		}
	}
	return configuration
}

func (configuration *NSConfiguration) FromNSUrl(url *tools.NSUrl) *NSConfiguration {
	var result NSConfiguration
	if configuration != nil {
		result = *configuration
	}
	result.ClientNetworkService = url.NsName
	result.NscInterfaceName = url.Intf
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
	result.ClientLabels = labels.String()
	return &result
}

func GetNamespace() string {
	namespace := getEnv(namespaceEnv, "Namespace", false)
	if len(namespace) == 0 {
		return "default"
	}
	return namespace
}
