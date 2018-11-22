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
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/dataplane/vppagent/pkg/memif"
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"strconv"
)

type MemifInterfaceConverter struct {
	*connection.Connection
	name      string
	ipAddress string
}

func NewMemifInterfaceConverter(c *connection.Connection, name string) Converter {
	rv := &MemifInterfaceConverter{
		Connection: c,
		name:       name,
	}
	return rv
}

func NewMemifInterfaceWithIpConverter(c *connection.Connection, name string, ipAddress string) Converter {
	rv := &MemifInterfaceConverter{
		Connection: c,
		name:       name,
		ipAddress:  ipAddress,
	}
	return rv
}

func (c *MemifInterfaceConverter) ToDataRequest(rv *rpc.DataRequest) (*rpc.DataRequest, error) {
	if rv == nil {
		rv = &rpc.DataRequest{}
	}

	isMaster, err := strconv.ParseBool(c.Mechanism.Parameters[connection.Master])
	if err != nil {
		isMaster = false
	}

	if err := memif.CreateDirectory(memif.BuildMemifDirectory(c.Mechanism)); err != nil {
		return nil, err
	}

	var ipAddresses []string
	if c.ipAddress != "" {
		ipAddresses = []string{c.ipAddress}
	}

	rv.Interfaces = append(rv.Interfaces, &interfaces.Interfaces_Interface{
		Name:        c.name,
		Type:        interfaces.InterfaceType_MEMORY_INTERFACE,
		Enabled:     true,
		IpAddresses: ipAddresses,
		Memif: &interfaces.Interfaces_Interface_Memif{
			Master:         isMaster,
			SocketFilename: memif.BuildSocketPath(c.Mechanism),
		},
	})
	return rv, nil
}
