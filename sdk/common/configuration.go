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
	"strconv"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	endpointNetworkServiceEnv = "ENDPOINT_NETWORK_SERVICE"
	endpointLabelsEnv         = "ENDPOINT_LABELS"
	clientNetworkServiceEnv   = "CLIENT_NETWORK_SERVICE"
	clientLabelsEnv           = "CLIENT_LABELS"
	tracerEnabledEnv          = "TRACER_ENABLED"
	mechanismTypeEnv          = "MECHANISM_TYPE"
	ipAddressEnv              = "IP_ADDRESS"
	routesEnv                 = "ROUTES"
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
	TracerEnabled          bool
	MechanismType          string
	IPAddress              string
	Routes             []string
}

// CompleteNSConfiguration fills all unset options from the env variables
func (nsc *NSConfiguration) CompleteNSConfiguration() {

	if nsc.NsmServerSocket == "" {
		nsc.NsmServerSocket = getEnv(nsmd.NsmServerSocketEnv, "nsmServerSocket", true)
	}

	if nsc.NsmClientSocket == "" {
		nsc.NsmClientSocket = getEnv(nsmd.NsmClientSocketEnv, "nsmClientSocket", true)
	}

	if nsc.Workspace == "" {
		nsc.Workspace = getEnv(nsmd.WorkspaceEnv, "workspace", true)
	}

	if nsc.EndpointNetworkService == "" {
		nsc.EndpointNetworkService = getEnv(endpointNetworkServiceEnv, "Advertise Network Service Name", false)
	}

	if nsc.ClientNetworkService == "" {
		nsc.ClientNetworkService = getEnv(clientNetworkServiceEnv, "Outgoing Network Service Name", false)
	}

	if nsc.EndpointLabels == "" {
		nsc.EndpointLabels = getEnv(endpointLabelsEnv, "Advertise labels", false)
	}

	if nsc.ClientLabels == "" {
		nsc.ClientLabels = getEnv(clientLabelsEnv, "Outgoing labels", false)
	}

	nsc.TracerEnabled, _ = strconv.ParseBool(getEnv(tracerEnabledEnv, "Tracer enabled", false))

	if configuration.MechanismType == "" {
		configuration.MechanismType = getEnv(mechanismTypeEnv, "Outgoing mechanism type", false)
	}

	if len(nsc.IPAddress) == 0 {
		nsc.IPAddress = getEnv(ipAddressEnv, "IP Address", false)
	}

	if len(configuration.Routes) == 0 {
		raw := getEnv(routesEnv, "Routes", false)
		if len(raw) > 1 {
			configuration.Routes = strings.Split(raw, ",")
		}
	}
}

func NSConfigurationFromUrl(configuration *NSConfiguration, url *tools.NSUrl) *NSConfiguration {
	var conf NSConfiguration
	if configuration != nil {
		conf = *configuration
	}
	conf.ClientNetworkService = url.NsName
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
	conf.ClientLabels = labels.String()
	return &conf
}

func NewNSConfigurationWithUrl(nsc *NSConfiguration, url *tools.NsUrl) *NSConfiguration {

	nsc = NewNSConfiguration(nsc)
	nsc.ClientNetworkService = url.NsName
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
	nsc.ClientLabels = labels.String()
	return nsc
}

func NewNSConfiguration(nsc *NSConfiguration) *NSConfiguration {
	if nsc == nil {
		nsc = &NSConfiguration{}
	}

	return nsc
}
