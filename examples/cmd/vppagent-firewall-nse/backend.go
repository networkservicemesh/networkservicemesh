// Copyright 2018 VMware, Inc.
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

package main

import (
	"context"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/sirupsen/logrus"
)

const (
	defaultVPPAgentEndpoint = "localhost:9112"
)

type vppagentBackend struct {
	vppAgentEndpoint string
	crossConnects    map[string]*crossconnect.CrossConnect
}

func (ns *vppagentBackend) New() error {
	ns.vppAgentEndpoint = defaultVPPAgentEndpoint
	ns.crossConnects = make(map[string]*crossconnect.CrossConnect)
	ns.Reset()
	return nil
}

func (ns *vppagentBackend) Request(ctx context.Context, incoming, outgoing *connection.Connection, baseDir string) error {

	crossConnectRequest := &crossconnect.CrossConnect{
		Id:      incoming.GetId(),
		Payload: "IP",
		Source: &crossconnect.CrossConnect_LocalSource{
			incoming,
		},
		Destination: &crossconnect.CrossConnect_LocalDestination{
			outgoing,
		},
	}

	_, dataChange, err := ns.CrossConnecVppInterfaces(ctx, crossConnectRequest, true, baseDir)
	if err != nil {
		logrus.Error(err)
		return err
	}

	// The Crossconnect converter generates and puts the Source Interface name here
	ingressIfName := dataChange.XCons[0].ReceiveInterface

	aclRules := map[string]string{
		"Allow ICMP":   "action=reflect,icmptype=8",
		"Allow TCP 80": "action=reflect,tcplowport=80,tcpupport=80",
	}

	err = ns.ApplyAclOnVppInterface(ctx, "IngressACL", ingressIfName, aclRules)
	if err != nil {
		logrus.Error(err)
		return err
	}

	return nil
}

func (ns *vppagentBackend) Close(ctx context.Context, conn *connection.Connection, baseDir string) error {
	// remove from connection
	crossConnectRequest, ok := ns.crossConnects[conn.GetId()]
	if ok {
		_, _, err := ns.CrossConnecVppInterfaces(ctx, crossConnectRequest, false, baseDir)
		if err != nil {
			logrus.Error(err)
			return err
		}
	}

	return nil
}

func (ns *vppagentBackend) GetMechanismType() connection.MechanismType {
	return connection.MechanismType_MEM_INTERFACE
}
