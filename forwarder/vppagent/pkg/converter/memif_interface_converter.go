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
	"os"
	"path"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"

	"github.com/pkg/errors"
	"go.ligato.io/vpp-agent/v3/proto/ligato/configurator"
	"go.ligato.io/vpp-agent/v3/proto/ligato/vpp"
	vpp_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"
	vpp_l3 "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/l3"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
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
	fullyQualifiedSocketFilename := path.Join(c.conversionParameters.BaseDir, memif.ToMechanism(c.Connection.GetMechanism()).GetSocketFilename())
	SocketDir := path.Dir(fullyQualifiedSocketFilename)

	var isMaster bool
	if c.conversionParameters.Side == DESTINATION {
		isMaster = c.conversionParameters.Terminate
	} else {
		isMaster = !c.conversionParameters.Terminate
	}

	if isMaster {
		if err := os.MkdirAll(SocketDir, os.ModePerm); err != nil {
			return nil, err
		}
	}

	var ipAddresses []string
	if c.conversionParameters.Terminate && c.conversionParameters.Side == DESTINATION {
		ipAddresses = []string{c.Connection.GetContext().GetIpContext().GetDstIpAddr()}
	}
	if c.conversionParameters.Terminate && c.conversionParameters.Side == SOURCE {
		ipAddresses = []string{c.Connection.GetContext().GetIpContext().GetSrcIpAddr()}
	}

	if c.conversionParameters.Name == "" {
		return nil, errors.New("ConnnectionConversionParameters.Name cannot be empty")
	}

	rv.VppConfig.Interfaces = append(rv.VppConfig.Interfaces, &vpp.Interface{
		Name:        c.conversionParameters.Name,
		Type:        vpp_interfaces.Interface_MEMIF,
		Enabled:     true,
		IpAddresses: ipAddresses,

		Link: &vpp_interfaces.Interface_Memif{
			Memif: &vpp_interfaces.MemifLink{
				Master:         isMaster,
				Mode:           vpp_interfaces.MemifLink_IP,
				SocketFilename: path.Join(fullyQualifiedSocketFilename),
			},
		},
	})

	// Process static routes

	routes := []*connectioncontext.Route{}
	switch c.conversionParameters.Side {
	case SOURCE:
		routes = c.Connection.GetContext().GetIpContext().GetDstRoutes()
	case DESTINATION:
		routes = c.Connection.GetContext().GetIpContext().GetSrcRoutes()
	}

	for _, route := range routes {
		route := &vpp.Route{
			Type:              vpp_l3.Route_INTER_VRF,
			DstNetwork:        route.Prefix,
			NextHopAddr:       extractCleanIPAddress(c.Connection.GetContext().GetIpContext().GetDstIpAddr()),
			OutgoingInterface: c.conversionParameters.Name,
		}
		rv.VppConfig.Routes = append(rv.VppConfig.Routes, route)
	}

	return rv, nil
}
