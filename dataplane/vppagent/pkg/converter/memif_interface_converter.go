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
	fmt "fmt"
	"os"
	"path"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
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

func (c *MemifInterfaceConverter) ToDataRequest(rv *rpc.DataRequest) (*rpc.DataRequest, error) {
	if rv == nil {
		rv = &rpc.DataRequest{}
	}
	fullyQualifiedSocketFilename := path.Join(c.conversionParameters.BaseDir, c.Connection.GetMechanism().GetSocketFilename())
	SocketDir := path.Dir(fullyQualifiedSocketFilename)
	if c.conversionParameters.Terminate {
		if err := os.MkdirAll(SocketDir, 0777); err != nil {
			return nil, err
		}
	}

	var ipAddresses []string
	if c.conversionParameters.Terminate && c.conversionParameters.Side == DESTINATION {
		ipAddresses = []string{c.Connection.GetContext()[connectioncontext.DstIpKey]}
	}
	if c.conversionParameters.Side == SOURCE {
		ipAddresses = []string{c.Connection.GetContext()[connectioncontext.SrcIpKey]}
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
			Master:         c.conversionParameters.Terminate,
			SocketFilename: path.Join(fullyQualifiedSocketFilename),
		},
	})
	return rv, nil
}
