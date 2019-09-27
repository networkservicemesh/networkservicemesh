// Copyright 2019 VMware, Inc.
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
	"net"
	"strconv"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	acl "github.com/ligato/vpp-agent/api/models/vpp/acl"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

const (
	action     = "action"     // DENY, PERMIT, REFLECT
	dstNet     = "dstnet"     // IPv4 or IPv6 CIDR
	srcNet     = "srcnet"     // IPv4 or IPv6 CIDR
	icmpType   = "icmptype"   // 8-bit unsigned integer
	tcpLowPort = "tcplowport" // 16-bit unsigned integer
	tcpUpPort  = "tcpupport"  // 16-bit unsigned integer
	udpLowPort = "udplowport" // 16-bit unsigned integer
	udpUpPort  = "udpupport"  // 16-bit unsigned integer
)

// ACL is a VPP Agent ACL composite
type ACL struct {
	Rules map[string]string
}

// Request implements the request handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//     ConnectionMap
func (a *ACL) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)

	ctx = WithConnectionMap(ctx) // Guarantees we will retrieve a non-nil Connectionmap from context.Context
	connectionMap := ConnectionMap(ctx)

	iface := connectionMap[request.GetConnection().GetId()]

	if iface == nil || iface.Name == "" {
		err := fmt.Errorf("found empty incoming connection name")
		return nil, err
	}

	err := a.appendDataChange(vppAgentConfig, iface.Name)
	if err != nil {
		return nil, err
	}

	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Request(ctx, request)
	}

	return request.GetConnection(), nil
}

// Close implements the close handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//     ConnectionMap
func (a *ACL) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)

	ctx = WithConnectionMap(ctx) // Guarantees we will retrieve a non-nil Connectionmap from context.Context
	connectionMap := ConnectionMap(ctx)

	iface := connectionMap[connection.GetId()]

	if iface == nil || iface.Name == "" {
		err := fmt.Errorf("found empty incoming connection name")
		return nil, err
	}

	err := a.appendDataChange(vppAgentConfig, iface.Name)
	if err != nil {
		return nil, err
	}
	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (a *ACL) Name() string {
	return "acl"
}

// NewACL creates an ACL
func NewACL(rules map[string]string) *ACL {
	return &ACL{
		Rules: rules,
	}
}

func (a *ACL) appendDataChange(rv *configurator.Config, ingressInterface string) error {
	if rv == nil {
		return fmt.Errorf("ACL.appendDataChange cannot be called with rv == nil")
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}
	if len(a.Rules) == 0 {
		return nil
	}

	rules := []*acl.ACL_Rule{}

	for _, rule := range a.Rules {
		parsed := tools.ParseKVStringToMap(rule, ",", "=")

		action, err := getAction(parsed)
		if err != nil {
			return fmt.Errorf("parsing rule %s failed with %v", rule, err)
		}

		match, err := getMatch(parsed)
		if err != nil {
			return fmt.Errorf("parsing rule %s failed with %v", rule, err)
		}

		match.Action = action
		rules = append(rules, match)
	}

	name := "ingress-acl-" + ingressInterface

	rv.VppConfig.Acls = append(rv.VppConfig.Acls, &acl.ACL{
		Name:  name,
		Rules: rules,
		Interfaces: &acl.ACL_Interfaces{
			Egress:  []string{},
			Ingress: []string{ingressInterface},
		},
	})

	return nil
}

func getAction(parsed map[string]string) (acl.ACL_Rule_Action, error) {
	actionName, ok := parsed[action]
	if !ok {
		return acl.ACL_Rule_Action(0), fmt.Errorf("rule should have 'action' set")
	}
	action, ok := acl.ACL_Rule_Action_value[strings.ToUpper(actionName)]
	if !ok {
		return acl.ACL_Rule_Action(0), fmt.Errorf("rule should have a valid 'action'")
	}
	return acl.ACL_Rule_Action(action), nil
}

func getIP(parsed map[string]string) (*acl.ACL_Rule_IpRule_Ip, error) {
	dstNet, dstNetOk := parsed[dstNet]
	srcNet, srcNetOk := parsed[srcNet]
	if dstNetOk {
		_, _, err := net.ParseCIDR(dstNet)
		if err != nil {
			return nil, fmt.Errorf("dstnet is not a valid CIDR [%v]. Failed with: %v", dstNet, err)
		}
	} else {
		dstNet = ""
	}

	if srcNetOk {
		_, _, err := net.ParseCIDR(srcNet)
		if err != nil {
			return nil, fmt.Errorf("srcnet is not a valid CIDR [%v]. Failed with: %v", srcNet, err)
		}
	} else {
		srcNet = ""
	}

	if dstNetOk || srcNetOk {
		return &acl.ACL_Rule_IpRule_Ip{
			DestinationNetwork: dstNet,
			SourceNetwork:      srcNet,
		}, nil
	}
	return nil, nil
}

func getICMP(parsed map[string]string) (*acl.ACL_Rule_IpRule_Icmp, error) {
	icmpType, ok := parsed[icmpType]
	if !ok {
		return nil, nil
	}
	icmpType8, err := strconv.ParseUint(icmpType, 10, 8)
	if err != nil {
		return nil, fmt.Errorf("failed parsing icmptype [%v] with: %v", icmpType, err)
	}
	return &acl.ACL_Rule_IpRule_Icmp{
		Icmpv6: false,
		IcmpCodeRange: &acl.ACL_Rule_IpRule_Icmp_Range{
			First: uint32(0),
			Last:  uint32(65535),
		},
		IcmpTypeRange: &acl.ACL_Rule_IpRule_Icmp_Range{
			First: uint32(icmpType8),
			Last:  uint32(icmpType8),
		},
	}, nil
}

func getPort(name string, parsed map[string]string) (uint16, bool, error) {
	port, ok := parsed[name]
	if !ok {
		return 0, false, nil
	}
	port16, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return 0, true, fmt.Errorf("failed parsing %s [%v] with: %v", name, port, err)
	}

	return uint16(port16), true, nil
}

func getTCP(parsed map[string]string) (*acl.ACL_Rule_IpRule_Tcp, error) {
	lowerPort, lpFound, lpErr := getPort(tcpLowPort, parsed)
	if !lpFound {
		return nil, nil
	} else if lpErr != nil {
		return nil, lpErr
	}

	upperPort, upFound, upErr := getPort(tcpUpPort, parsed)
	if !upFound {
		return nil, nil
	} else if upErr != nil {
		return nil, lpErr
	}

	return &acl.ACL_Rule_IpRule_Tcp{
		DestinationPortRange: &acl.ACL_Rule_IpRule_PortRange{
			LowerPort: uint32(lowerPort),
			UpperPort: uint32(upperPort),
		},
		SourcePortRange: &acl.ACL_Rule_IpRule_PortRange{
			LowerPort: uint32(0),
			UpperPort: uint32(65535),
		},
		TcpFlagsMask:  0,
		TcpFlagsValue: 0,
	}, nil
}

func getUDP(parsed map[string]string) (*acl.ACL_Rule_IpRule_Udp, error) {
	lowerPort, lpFound, lpErr := getPort(udpLowPort, parsed)
	if !lpFound {
		return nil, nil
	} else if lpErr != nil {
		return nil, lpErr
	}

	upperPort, upFound, upErr := getPort(udpUpPort, parsed)
	if !upFound {
		return nil, nil
	} else if upErr != nil {
		return nil, lpErr
	}

	return &acl.ACL_Rule_IpRule_Udp{
		DestinationPortRange: &acl.ACL_Rule_IpRule_PortRange{
			LowerPort: uint32(lowerPort),
			UpperPort: uint32(upperPort),
		},
		SourcePortRange: &acl.ACL_Rule_IpRule_PortRange{
			LowerPort: uint32(0),
			UpperPort: uint32(65535),
		},
	}, nil
}

func getIPRule(parsed map[string]string) (*acl.ACL_Rule_IpRule, error) {
	ip, err := getIP(parsed)
	if err != nil {
		return nil, err
	}

	icmp, err := getICMP(parsed)
	if err != nil {
		return nil, err
	}

	tcp, err := getTCP(parsed)
	if err != nil {
		return nil, err
	}

	udp, err := getUDP(parsed)
	if err != nil {
		return nil, err
	}

	return &acl.ACL_Rule_IpRule{
		Ip:   ip,
		Icmp: icmp,
		Tcp:  tcp,
		Udp:  udp,
	}, nil
}

func getMatch(parsed map[string]string) (*acl.ACL_Rule, error) {
	ipRule, err := getIPRule(parsed)
	if err != nil {
		return nil, err
	}

	return &acl.ACL_Rule{
		IpRule:    ipRule,
		MacipRule: nil,
	}, nil
}
