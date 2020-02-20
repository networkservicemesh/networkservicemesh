// Copyright (c) 2019 Doc.ai and/or its affiliates.
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
	"fmt"

	sdk "github.com/networkservicemesh/networkservicemesh/forwarder/sdk/wireguard"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	vpp_l2 "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/l2"

	vpp_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"

	"go.ligato.io/vpp-agent/v3/proto/ligato/vpp"

	"github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/kernelforwarder/remote"

	"go.ligato.io/vpp-agent/v3/proto/ligato/configurator"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/wireguard"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

type wgInterfaces struct{ *sdk.DeviceManager }

func (w *wgInterfaces) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	if crossConnect.GetSource().GetMechanism().GetType() == wireguard.MECHANISM ||
		crossConnect.GetDestination().GetMechanism().GetType() == wireguard.MECHANISM {
		c := DataChange(ctx)
		if crossConnect.GetSource().IsRemote() && !crossConnect.GetDestination().IsRemote() {
			err := w.appendInterfaces(c, crossConnect.Id, crossConnect.GetSource(), remote.INCOMING)
			if err != nil {
				return nil, err
			}
		} else if !crossConnect.GetSource().IsRemote() && crossConnect.GetDestination().IsRemote() {
			err := w.appendInterfaces(c, crossConnect.Id, crossConnect.GetDestination(), remote.OUTGOING)
			if err != nil {
				return nil, err
			}
		}
	}
	next := Next(ctx)
	if next == nil {
		return crossConnect, nil
	}
	return next.Request(ctx, crossConnect)
}

func (w *wgInterfaces) Close(ctx context.Context, crossConn *crossconnect.CrossConnect) (*empty.Empty, error) {
	if crossConn.GetLocalSource().GetMechanism().GetType() == wireguard.MECHANISM ||
		crossConn.GetLocalDestination().GetMechanism().GetType() == wireguard.MECHANISM {
		if crossConn.GetSource().IsRemote() && !crossConn.GetDestination().IsRemote() {
			w.DeleteWireguardDevice(w.getWGDeviceName(crossConn.Id, false))
		} else if !crossConn.GetSource().IsRemote() && crossConn.GetDestination().IsRemote() {
			w.DeleteWireguardDevice(w.getWGDeviceName(crossConn.Id, true))
		}
	}
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConn)
}

// NewWgInterfaces creates chain element for manage wirguard mechanism cases
func NewWgInterfaces() forwarder.ForwarderServer {
	return &wgInterfaces{
		DeviceManager: sdk.NewWireguardDeviceManager(true),
	}
}

func (w *wgInterfaces) appendInterfaces(rv *configurator.Config, id string, r *connection.Connection, dst uint8) error {
	name := "SRC-" + id
	if dst == remote.INCOMING {
		name = "DST-" + id
	}
	wgName := w.getWGDeviceName(id, dst != remote.INCOMING)
	vppWgName := "VPP-" + wgName
	err := w.CreateWireguardDevice(wgName, r, dst == remote.INCOMING)
	if err != nil {
		logrus.Errorf("common: failed to create %q, %v", wgName, err)
		return err
	}
	link, err := netlink.LinkByName(wgName)
	if err != nil {
		logrus.Errorf("common: failed to lookup %q, %v", wgName, err)
		return err
	}
	if err = netlink.LinkSetUp(link); err != nil {
		logrus.Errorf("common: failed to bring %q up: %v", wgName, err)
		return err
	}
	if err != nil {
		w.DeleteWireguardDevice(wgName)
		return err
	}
	rv.VppConfig.Interfaces = append(rv.VppConfig.Interfaces, &vpp.Interface{
		Name: vppWgName,
		Type: vpp_interfaces.Interface_AF_PACKET,
		Link: &vpp_interfaces.Interface_Afpacket{
			Afpacket: &vpp_interfaces.AfpacketLink{
				HostIfName: wgName,
			},
		},
		Enabled: true,
	})
	rv.VppConfig.XconnectPairs = append(rv.VppConfig.XconnectPairs, &vpp_l2.XConnectPair{
		ReceiveInterface:  vppWgName,
		TransmitInterface: name,
	})
	rv.VppConfig.XconnectPairs = append(rv.VppConfig.XconnectPairs, &vpp_l2.XConnectPair{
		ReceiveInterface:  name,
		TransmitInterface: vppWgName,
	})
	return nil
}

func (w *wgInterfaces) getWGDeviceName(id string, src bool) string {
	prefix := "SRC"
	if !src {
		prefix = "DST"
	}
	return fmt.Sprintf("WG-%v-%v", prefix, id)
}
