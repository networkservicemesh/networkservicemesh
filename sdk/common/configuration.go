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
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
	viperConfig            *viper.Viper
}

func (nsc *NSConfiguration) getEnv(key, description string, mandatory bool) string {

	value := nsc.viperConfig.GetString(key)
	if value == "" {
		if mandatory {
			logrus.Fatalf("Error getting %v", key)
		} else {
			logrus.Infof("%v not found.", key)
			return ""
		}
	}
	logrus.Infof("%s: %s", description, value)
	return value
}

// CompleteNSConfiguration fills all unset options from the env variables
func (nsc *NSConfiguration) completeNSConfiguration() {

	if nsc.NsmServerSocket == "" {
		nsc.NsmServerSocket = nsc.getEnv(nsmd.NsmServerSocketEnv, "nsmServerSocket", true)
	}

	if nsc.NsmClientSocket == "" {
		nsc.NsmClientSocket = nsc.getEnv(nsmd.NsmClientSocketEnv, "nsmClientSocket", true)
	}

	if nsc.Workspace == "" {
		nsc.Workspace = nsc.getEnv(nsmd.WorkspaceEnv, "workspace", true)
	}

	if nsc.EndpointNetworkService == "" {
		nsc.EndpointNetworkService = nsc.getEnv(endpointNetworkServiceEnv, "Advertise Network Service Name", false)
	}

	if nsc.ClientNetworkService == "" {
		nsc.ClientNetworkService = nsc.getEnv(clientNetworkServiceEnv, "Outgoing Network Service Name", false)
	}

	if nsc.EndpointLabels == "" {
		nsc.EndpointLabels = nsc.getEnv(endpointLabelsEnv, "Advertise labels", false)
	}

	if nsc.ClientLabels == "" {
		nsc.ClientLabels = nsc.getEnv(clientLabelsEnv, "Outgoing labels", false)
	}

	nsc.TracerEnabled, _ = strconv.ParseBool(nsc.getEnv(tracerEnabledEnv, "Tracer enabled", false))

	if nsc.MechanismType == "" {
		nsc.MechanismType = nsc.getEnv(mechanismTypeEnv, "Outgoing mechanism type", false)
	}

	if nsc.IPAddress == "" {
		nsc.IPAddress = nsc.getEnv(ipAddressEnv, "IP Address", false)
	}

	if len(configuration.Routes) == 0 {
		raw := getEnv(routesEnv, "Routes", false)
		if len(raw) > 1 {
			configuration.Routes = strings.Split(raw, ",")
		}
	}
}

func (nsc *NSConfiguration) bindNSConfiguration() {
	err := nsc.viperConfig.BindEnv(nsmd.NsmServerSocketEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", nsmd.NsmServerSocketEnv)
	}
	err = nsc.viperConfig.BindEnv(nsmd.NsmClientSocketEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", nsmd.NsmClientSocketEnv)
	}
	err = nsc.viperConfig.BindEnv(nsmd.WorkspaceEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", nsmd.WorkspaceEnv)
	}
	err = nsc.viperConfig.BindEnv(endpointNetworkServiceEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", endpointNetworkServiceEnv)
	}
	err = nsc.viperConfig.BindEnv(clientNetworkServiceEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", clientNetworkServiceEnv)
	}
	err = nsc.viperConfig.BindEnv(endpointLabelsEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", endpointLabelsEnv)
	}
	err = nsc.viperConfig.BindEnv(clientLabelsEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", clientLabelsEnv)
	}
	err = nsc.viperConfig.BindEnv(tracerEnabledEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", tracerEnabledEnv)
	}
	err = nsc.viperConfig.BindEnv(ipAddressEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", ipAddressEnv)
	}
	err = nsc.viperConfig.BindEnv(ipAddressEnv)
	if err != nil {
		logrus.Errorf("Unable to bind %s", ipAddressEnv)
	}
}

// NewNSConfigurationWithURL ensure the new NS configuration with a provided URL
func NewNSConfigurationWithURL(nsc *NSConfiguration, url *tools.NSUrl) *NSConfiguration {

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

// NewNSConfiguration ensure the new NS configuration
func NewNSConfiguration(nsc *NSConfiguration) *NSConfiguration {
	if nsc == nil {
		nsc = &NSConfiguration{}
	}

	if nsc.viperConfig == nil {
		nsc.viperConfig = viper.New()
		nsc.bindNSConfiguration()
		nsc.completeNSConfiguration()
	}

	return nsc
}
