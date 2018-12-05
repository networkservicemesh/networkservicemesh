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
	"github.com/ligato/vpp-agent/plugins/linux/model/l3"
	"os"
	"path"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
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

func (c *MemifInterfaceConverter) ToDataRequest(rv *rpc.DataRequest, connect bool) (*rpc.DataRequest, error) {
	if rv == nil {
		rv = &rpc.DataRequest{}
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

	rv.Interfaces = append(rv.Interfaces, &interfaces.Interfaces_Interface{
		Name:        c.conversionParameters.Name,
		Type:        interfaces.InterfaceType_MEMORY_INTERFACE,
		Enabled:     true,
		IpAddresses: ipAddresses,
		Memif: &interfaces.Interfaces_Interface_Memif{
			Master:         isMaster,
			SocketFilename: path.Join(fullyQualifiedSocketFilename),
		},
	})

	// Process static routes
	if c.conversionParameters.Side == SOURCE {
		m := c.GetMechanism()
		filepath, err := m.NetNsFileName()
		for _, route := range c.Connection.GetContext().GetRoutes() {
			route := &l3.LinuxStaticRoutes_Route{
				Name:        "route_" + TempIfName(),
				DstIpAddr:   appendNetmaskIfNeeded(route.Prefix),
				Description: "Route to " + route.Prefix,
				Interface:   c.conversionParameters.Name,
				GwAddr: extractCleanIPAddress(c.Connection.GetContext().DstIpAddr),
			}
			if err == nil {
				route.Namespace = &l3.LinuxStaticRoutes_Route_Namespace{
					Type:     l3.LinuxStaticRoutes_Route_Namespace_FILE_REF_NS,
					Filepath: filepath,
				}
			}
			rv.LinuxRoutes = append(rv.LinuxRoutes, route)
		}
	}
	return rv, nil
}
