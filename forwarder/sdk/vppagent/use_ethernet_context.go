// Copyright (c) 2019 Cisco and/or its affiliates.
//
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

package vppagent

import (
	"context"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/forwarder/vppagent/pkg/converter"

	"github.com/ligato/vpp-agent/api/models/linux"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

//UseEthernetContext fills ethernet context for dst interface if it is empty
func UseEthernetContext() forwarder.ForwarderServer {
	return &ethernetContextUpdater{}
}

type ethernetContextUpdater struct {
}

func (c *ethernetContextUpdater) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	resp, err := next.Request(ctx, crossConnect)
	if err == nil {
		setEthernetContext(ctx, crossConnect)
	}
	return resp, err
}

func (c *ethernetContextUpdater) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConnect)
}

func setEthernetContext(ctx context.Context, c *crossconnect.CrossConnect) {
	if c.GetLocalDestination() != nil && !c.GetLocalDestination().GetContext().IsEthernetContextEmtpy() {
		return
	}
	if c.GetLocalSource() != nil && !c.GetLocalSource().GetContext().IsEthernetContextEmtpy() {
		return
	}
	mac := getVppDestinationInterfaceMacByID(ctx, c.Id)
	if mac == "" {
		Logger(ctx).Warn("DST mac is empty")
		return
	}
	if c.GetLocalDestination() != nil {
		c.GetLocalDestination().GetContext().EthernetContext = &connectioncontext.EthernetContext{
			DstMac: mac,
		}
	}
	if c.GetLocalSource() != nil {
		dataChange := DataChange(ctx)
		dataChange.LinuxConfig.ArpEntries = append(dataChange.LinuxConfig.ArpEntries, &linux.ARPEntry{
			IpAddress: strings.Split(c.GetLocalSource().GetContext().IpContext.DstIpAddr, "/")[0],
			Interface: converter.GetSrcInterfaceName(c.Id),
			HwAddress: mac,
		})
		_, err := ConfiguratorClient(ctx).Update(ctx, &configurator.UpdateRequest{Update: dataChange})
		if err != nil {
			Logger(ctx).Errorf("An error during update arp entries: %v", err.Error())
		}
	}
}

func getVppDestinationInterfaceMacByID(ctx context.Context, id string) string {
	dstName := converter.GetDstInterfaceName(id)
	dumpResp := dumpRequest(ctx)
	if dumpResp == nil {
		return ""
	}
	for _, iface := range dumpResp.LinuxConfig.Interfaces {
		if iface.Name == dstName {
			return iface.PhysAddress
		}
	}
	return ""
}

func dumpRequest(ctx context.Context) *configurator.Config {
	client := ConfiguratorClient(ctx)
	if client == nil {
		Logger(ctx).Warn("Configuration client is empty, can not request dump")
		return nil
	}
	dumpResp, err := client.Dump(context.Background(), &configurator.DumpRequest{})
	if err != nil {
		Logger(ctx).Errorf("An error during client.Dump: %v", err)
		return nil
	}
	Logger(ctx).Infof("Dump response: %v", dumpResp.String())
	return dumpResp.Dump
}
