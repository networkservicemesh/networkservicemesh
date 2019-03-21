// Copyright (c) 2018 Cisco and/or its affiliates.
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

package converter

import (
	"fmt"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	"github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/ligato/vpp-agent/api/models/vpp/l3"
	"os"
	"path"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
)

type MemifInterfaceConverter struct {
	*connection.Connection
	conversionParameters *ConnectionConversionParameters
}

func NewMemifInterfaceConverter(c *connection.Connection, conversionParameters *ConnectionConversionParameters) Converter {
	rv := &MemifInterfaceConverter{
		Connection:           c,
		conversionParameters: conversionParameters,
	}
	return rv
}

func (c *MemifInterfaceConverter) ToDataRequest(rv *configurator.Config, connect bool) (*configurator.Config, error) {
	if rv == nil {
		rv = &configurator.Config{}
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}
	fullyQualifiedSocketFilename := path.Join(c.conversionParameters.BaseDir, c.Connection.GetMechanism().GetSocketFilename())
	SocketDir := path.Dir(fullyQualifiedSocketFilename)

	var isMaster bool
	if c.conversionParameters.Side == DESTINATION {
		isMaster = c.conversionParameters.Terminate
	} else {
		isMaster = !c.conversionParameters.Terminate
	}

	if isMaster {
		if err := os.MkdirAll(SocketDir, 0777); err != nil {
			return nil, err
		}
	}

	var ipAddresses []string
	if c.conversionParameters.Terminate && c.conversionParameters.Side == DESTINATION {
		ipAddresses = []string{c.Connection.GetContext().DstIpAddr}
	}
	if c.conversionParameters.Terminate && c.conversionParameters.Side == SOURCE {
		ipAddresses = []string{c.Connection.GetContext().SrcIpAddr}
	}

	if c.conversionParameters.Name == "" {
		return nil, fmt.Errorf("ConnnectionConversionParameters.Name cannot be empty")
	}

	rv.VppConfig.Interfaces = append(rv.VppConfig.Interfaces, &vpp.Interface{
		Name:        c.conversionParameters.Name,
		Type:        vpp_interfaces.Interface_MEMIF,
		Enabled:     true,
		IpAddresses: ipAddresses,
		Link: &vpp_interfaces.Interface_Memif{
			Memif: &vpp_interfaces.MemifLink{
				Master: isMaster,
				SocketFilename: path.Join(fullyQualifiedSocketFilename),
			},
		},
	})

	// Process static routes
	if c.conversionParameters.Side == SOURCE {
		for _, route := range c.Connection.GetContext().GetRoutes() {
			route := &vpp.Route{
				Type: vpp_l3.Route_INTER_VRF,
				DstNetwork:         route.Prefix,
				//Description:       "Route to " + route.Prefix,
				NextHopAddr:       extractCleanIPAddress(c.Connection.GetContext().DstIpAddr),
				OutgoingInterface: c.conversionParameters.Name,
			}
			rv.VppConfig.Routes = append(rv.VppConfig.Routes, route)
		}
	}
	return rv, nil
}
