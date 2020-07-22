// Copyright (c) 2020 Doc.ai and/or its affiliates.
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
	"strings"

	vpp_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"

	vpp_l2 "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/l2"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"

	"go.ligato.io/vpp-agent/v3/proto/ligato/vpp"

	"github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/kernelforwarder/remote"

	"go.ligato.io/vpp-agent/v3/proto/ligato/configurator"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/wireguard"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
)

type wgInterfaces struct{}

func (w *wgInterfaces) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	if crossConnect.GetSource().GetMechanism().GetType() == wireguard.MECHANISM ||
		crossConnect.GetDestination().GetMechanism().GetType() == wireguard.MECHANISM {
		c := DataChange(ctx)
		if crossConnect.GetSource().IsRemote() && !crossConnect.GetDestination().IsRemote() {
			err := w.appendInterfaces(c, crossConnect.Id, crossConnect.GetSource(), remote.INCOMING)
			if err != nil {
				panic(err.Error())
				return nil, err
			}
		} else if !crossConnect.GetSource().IsRemote() && crossConnect.GetDestination().IsRemote() {
			err := w.appendInterfaces(c, crossConnect.Id, crossConnect.GetDestination(), remote.OUTGOING)
			if err != nil {
				panic(err.Error())
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
	// TODO: add close logic here
	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConn)
}

// NewWgInterfaces creates chain element for manage wirguard mechanism cases
func NewWgInterfaces() forwarder.ForwarderServer {
	return &wgInterfaces{}
}

func (w *wgInterfaces) appendInterfaces(rv *configurator.Config, id string, r *connection.Connection, dst uint8) error {
	//TODO: probably we need to add more configurations here:
	name := "SRC-" + id
	if dst == remote.INCOMING {
		name = "DST-" + id
	}
	wgName := w.getWGDeviceName(id, dst != remote.INCOMING)
	vppWgName := "VPP-" + wgName

	mech := wireguard.ToMechanism(r.Mechanism)
	var (
		remoteIP   string
		srcIP      string
		privateKey string
		publicKey  string
		port       int
		remotePort int
		err        error
	)
	if dst == remote.INCOMING {
		privateKey, err = mech.DstPrivateKey()
		if err != nil {
			return err
		}
		publicKey, err = mech.SrcPublicKey()
		if err != nil {
			return err
		}
		port, err = mech.SrcPort()
		if err != nil {
			return err
		}
		remotePort, err = mech.DstPort()
		if err != nil {
			return err
		}
		srcIP = r.Context.IpContext.DstIpAddr
		remoteIP = r.Context.IpContext.SrcIpAddr
	} else {
		privateKey, err = mech.SrcPrivateKey()
		if err != nil {
			return err
		}
		publicKey, err = mech.DstPublicKey()
		if err != nil {
			return err
		}
		port, err = mech.DstPort()
		if err != nil {
			return err
		}
		remotePort, err = mech.SrcPort()
		if err != nil {
			return err
		}
		srcIP = r.Context.IpContext.SrcIpAddr
		remoteIP = r.Context.IpContext.DstIpAddr
	}

	rv.VppConfig.WgDevice = &vpp.WgDevice{
		PrivateKey: privateKey,
		Port:       uint32(port),
	}

	rv.VppConfig.WgPeers = append(rv.VppConfig.WgPeers, &vpp.WgPeer{
		PublicKey:           publicKey,
		Endpoint:            strings.Split(remoteIP, "/")[0],
		TunInterface:        vppWgName,
		Port:                uint32(remotePort),
		PersistentKeepalive: 10,
		AllowedIp:           strings.Split(remoteIP, "/")[0],
	})
	rv.VppConfig.Interfaces = append(rv.VppConfig.Interfaces, &vpp.Interface{
		Name:        vppWgName,
		IpAddresses: []string{srcIP},
		Enabled:     true,
		Link: &vpp_interfaces.Interface_Ipip{Ipip: &vpp_interfaces.IPIPLink{
			TunnelMode: 0,
			SrcAddr:    strings.Split(srcIP, "/")[0],
			DstAddr:    strings.Split(remoteIP, "/")[0],
		}},
		Type: vpp_interfaces.Interface_IPIP_TUNNEL,
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
