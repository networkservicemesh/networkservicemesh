// Copyright (c) 2019 Cisco and/or its affiliates.
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

package forwarder

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

//NewEthernetContextUpdater fills ethernet context for dst interface if it is empty
func NewEthernetContextUpdater() forwarder.ForwarderServer {
	return &ethernetContextUpdater{}
}

type ethernetContextUpdater struct {
}

func (c *ethernetContextUpdater) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	updateEthernetContext(ctx, crossConnect)
	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	return next.Request(ctx, crossConnect)
}

func (c *ethernetContextUpdater) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	updateEthernetContext(ctx, crossConnect)
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConnect)
}

func updateEthernetContext(ctx context.Context, c *crossconnect.CrossConnect) {
	//need to update ethernet context after problem https://github.com/ligato/vpp-agent/issues/1525 will be solved
	getVppDestentaionInterfaceMacByInterfaceName(ctx, c)
}

func getVppDestentaionInterfaceMacByInterfaceName(ctx context.Context, _ *crossconnect.CrossConnect) string {
	client := ConfiguratorClient(ctx)
	dumpResp, err := client.Dump(context.Background(), &configurator.DumpRequest{})
	if err != nil {
		Logger(ctx).Errorf("And error during client.Dump: %v", err)
	} else {
		Logger(ctx).Infof("Dump response: %v", dumpResp.String())
	}
	getResp, err := client.Get(ctx, &configurator.GetRequest{})
	if err != nil {
		Logger(ctx).Errorf("And error during client.Get: %v", err)
	} else {
		Logger(ctx).Infof("Get response: %v", getResp.String())
	}
	return ""
}
