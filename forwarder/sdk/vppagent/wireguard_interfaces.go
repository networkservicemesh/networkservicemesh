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

	"github.com/golang/protobuf/ptypes/empty"
	"go.ligato.io/vpp-agent/v3/proto/ligato/configurator"
	"go.ligato.io/vpp-agent/v3/proto/ligato/vpp"
	vpp_interfaces "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/interfaces"
	vpp_l3 "go.ligato.io/vpp-agent/v3/proto/ligato/vpp/l3"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/wireguard"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/api/forwarder"
	"github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/kernelforwarder/remote"
)

const (
	// Persistent keepalive interval (sec)
	defaultPersistentKeepalive = 20
)

type wgInterfaces struct {
	direction uint8
}

func (w *wgInterfaces) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	if crossConnect.GetSource().GetMechanism().GetType() == wireguard.MECHANISM ||
		crossConnect.GetDestination().GetMechanism().GetType() == wireguard.MECHANISM {
		c := DataChange(ctx)
		if crossConnect.GetSource().IsRemote() && !crossConnect.GetDestination().IsRemote() {
			w.direction = remote.INCOMING
			err := w.appendInterfaces(c, crossConnect.Id, crossConnect.GetSource(), w.direction)
			if err != nil {
				return nil, err
			}
		} else if !crossConnect.GetSource().IsRemote() && crossConnect.GetDestination().IsRemote() {
			w.direction = remote.OUTGOING
			err := w.appendInterfaces(c, crossConnect.Id, crossConnect.GetDestination(), w.direction)
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
	c := DataChange(ctx)
	if w.direction == remote.INCOMING {
		if err := w.appendInterfaces(c, crossConn.Id, crossConn.GetSource(), w.direction); err != nil {
			return nil, err
		}
	} else if w.direction == remote.OUTGOING {
		if err := w.appendInterfaces(c, crossConn.Id, crossConn.GetDestination(), w.direction); err != nil {
			return nil, err
		}
	}
	w.direction = ^uint8(0)

	next := Next(ctx)
	if next == nil {
		return new(empty.Empty), nil
	}
	return next.Close(ctx, crossConn)
}

// NewWgInterfaces creates chain element for manage wireguard mechanism cases
// Direction is undefined by default (^uint8(0))
func NewWgInterfaces() forwarder.ForwarderServer {
	return &wgInterfaces{
		direction: ^uint8(0),
	}
}

func (w *wgInterfaces) appendInterfaces(rv *configurator.Config, id string, r *connection.Connection, direction uint8) error {
	name := "SRC-" + id
	if direction == remote.INCOMING {
		name = "DST-" + id
	}
	vppWgInterfaceName := w.getVppWgInterfaceName(id, direction != remote.INCOMING)

	mechanism := wireguard.ToMechanism(r.GetMechanism())

	/* Create interface */
	var (
		localPrivateKey   string
		remotePublicKey   string
		localPort         int
		remotePort        int
		srcIP             string
		remoteIP          string
		wireguardSrcIP    string
		wireguardRemoteIP string
		err               error
	)
	if direction == remote.INCOMING {
		if localPrivateKey, err = mechanism.DstPrivateKey(); err != nil {
			return err
		}
		if remotePublicKey, err = mechanism.SrcPublicKey(); err != nil {
			return err
		}
		if localPort, err = mechanism.DstPort(); err != nil {
			return err
		}
		if remotePort, err = mechanism.SrcPort(); err != nil {
			return err
		}
		if srcIP, err = mechanism.DstIP(); err != nil {
			return err
		}
		if remoteIP, err = mechanism.SrcIP(); err != nil {
			return err
		}

		wireguardSrcIP = r.GetContext().GetIpContext().GetDstIpAddr()
		wireguardRemoteIP = r.GetContext().GetIpContext().GetSrcIpAddr()
	} else {
		if localPrivateKey, err = mechanism.SrcPrivateKey(); err != nil {
			return err
		}
		if remotePublicKey, err = mechanism.DstPublicKey(); err != nil {
			return err
		}
		if localPort, err = mechanism.SrcPort(); err != nil {
			return err
		}
		if remotePort, err = mechanism.DstPort(); err != nil {
			return err
		}
		if srcIP, err = mechanism.SrcIP(); err != nil {
			return err
		}
		if remoteIP, err = mechanism.DstIP(); err != nil {
			return err
		}

		wireguardSrcIP = r.GetContext().GetIpContext().GetSrcIpAddr()
		wireguardRemoteIP = r.GetContext().GetIpContext().GetDstIpAddr()
	}

	rv.VppConfig.Interfaces = append(rv.VppConfig.Interfaces, &vpp.Interface{
		Name:        vppWgInterfaceName,
		IpAddresses: []string{wireguardSrcIP},
		Enabled:     true,
		Link: &vpp_interfaces.Interface_Wireguard{Wireguard: &vpp_interfaces.WireguardLink{
			PrivateKey: localPrivateKey,
			Port:       uint32(localPort),
			SrcAddr:    strings.Split(srcIP, "/")[0],
		}},
		Type: vpp_interfaces.Interface_WIREGUARD_TUNNEL,
	})

	rv.VppConfig.WgPeers = append(rv.VppConfig.WgPeers, &vpp.WgPeer{
		PublicKey:           remotePublicKey,
		Endpoint:            strings.Split(remoteIP, "/")[0],
		WgIfName:            vppWgInterfaceName,
		Port:                uint32(remotePort),
		PersistentKeepalive: defaultPersistentKeepalive,
		AllowedIps:          []string{wireguardRemoteIP},
	})

	/* Create L3Xconnects */
	rv.VppConfig.L3Xconnects = append(rv.VppConfig.L3Xconnects, &vpp_l3.L3XConnect{
		Interface: vppWgInterfaceName,
		Protocol:  vpp_l3.L3XConnect_IPV4,
		Paths: []*vpp_l3.L3XConnect_Path{
			{
				OutgoingInterface: name,
				NextHopAddr:       srcIP,
			},
		},
	})
	rv.VppConfig.L3Xconnects = append(rv.VppConfig.L3Xconnects, &vpp_l3.L3XConnect{
		Interface: name,
		Protocol:  vpp_l3.L3XConnect_IPV4,
		Paths: []*vpp_l3.L3XConnect_Path{
			{
				OutgoingInterface: vppWgInterfaceName,
				NextHopAddr:       remoteIP,
			},
		},
	})
	return nil
}

func (w *wgInterfaces) getVppWgInterfaceName(id string, src bool) string {
	prefix := "SRC"
	if !src {
		prefix = "DST"
	}
	return "VPP" + fmt.Sprintf("WG-%v-%v", prefix, id)
}
