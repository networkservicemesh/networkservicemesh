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

package converter

import (
	"fmt"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/vpp-agent/plugins/vpp/model/acl"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
)

type aclConverter struct {
	Name             string
	Rules            map[string]string
	IngressInterface string
}

// NewAclConverter creates a new ACL converter
//
// action - DENY, PERMIT, REFLECT
//
// dtsnet, srcnet - IPv4 or IPv6 CIDR
//
// icmptype - 8-bit unsigned integer
//
// tcplowport, tcpupport - 16-bit unsigned integer
//
// udplowport, udpupport - 16-bit unsigned integer
//
func NewAclConverter(name, ingress string, rules map[string]string) Converter {
	rv := &aclConverter{
		Name:             name,
		Rules:            rules,
		IngressInterface: ingress,
	}
	return rv
}

func getAction(parsed map[string]string) (acl.AclAction, error) {
	action_name, ok := parsed["action"]
	if !ok {
		return acl.AclAction(0), fmt.Errorf("Rule should have 'action' set.")
	}
	action, ok := acl.AclAction_value[strings.ToUpper(action_name)]
	if !ok {
		return acl.AclAction(0), fmt.Errorf("Rule should have a valid 'action'.")
	}
	return acl.AclAction(action), nil
}

func getIp(parsed map[string]string) (*acl.AccessLists_Acl_Rule_Match_IpRule_Ip, error) {
	dstNet, dstNetOk := parsed["dstnet"]
	srcNet, srcNetOk := parsed["srcnet"]
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
		return &acl.AccessLists_Acl_Rule_Match_IpRule_Ip{
			DestinationNetwork: dstNet,
			SourceNetwork:      srcNet,
		}, nil
	}
	return nil, nil
}

func getIcmp(parsed map[string]string) (*acl.AccessLists_Acl_Rule_Match_IpRule_Icmp, error) {
	icmpType, ok := parsed["icmptype"]
	if !ok {
		return nil, nil
	}
	icmpType8, err := strconv.ParseUint(icmpType, 10, 8)
	if err != nil {
		return nil, fmt.Errorf("Failed parsing icmptype [%v] with: %v", icmpType, err)
	}
	return &acl.AccessLists_Acl_Rule_Match_IpRule_Icmp{
		Icmpv6: false,
		IcmpCodeRange: &acl.AccessLists_Acl_Rule_Match_IpRule_Icmp_Range{
			First: uint32(0),
			Last:  uint32(65535),
		},
		IcmpTypeRange: &acl.AccessLists_Acl_Rule_Match_IpRule_Icmp_Range{
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
		return 0, true, fmt.Errorf("Failed parsing %s [%v] with: %v", name, port, err)
	}

	return uint16(port16), true, nil
}

func getTcp(parsed map[string]string) (*acl.AccessLists_Acl_Rule_Match_IpRule_Tcp, error) {
	lowerPort, lpFound, lpErr := getPort("tcplowport", parsed)
	if !lpFound {
		return nil, nil
	} else if lpErr != nil {
		return nil, lpErr
	}

	upperPort, upFound, upErr := getPort("tcpupport", parsed)
	if !upFound {
		return nil, nil
	} else if upErr != nil {
		return nil, lpErr
	}

	return &acl.AccessLists_Acl_Rule_Match_IpRule_Tcp{
		DestinationPortRange: &acl.AccessLists_Acl_Rule_Match_IpRule_PortRange{
			LowerPort: uint32(lowerPort),
			UpperPort: uint32(upperPort),
		},
		SourcePortRange: &acl.AccessLists_Acl_Rule_Match_IpRule_PortRange{
			LowerPort: uint32(0),
			UpperPort: uint32(65535),
		},
		TcpFlagsMask:  0,
		TcpFlagsValue: 0,
	}, nil
}

func getUdp(parsed map[string]string) (*acl.AccessLists_Acl_Rule_Match_IpRule_Udp, error) {
	lowerPort, lpFound, lpErr := getPort("udplowport", parsed)
	if !lpFound {
		return nil, nil
	} else if lpErr != nil {
		return nil, lpErr
	}

	upperPort, upFound, upErr := getPort("udpupport", parsed)
	if !upFound {
		return nil, nil
	} else if upErr != nil {
		return nil, lpErr
	}

	return &acl.AccessLists_Acl_Rule_Match_IpRule_Udp{
		DestinationPortRange: &acl.AccessLists_Acl_Rule_Match_IpRule_PortRange{
			LowerPort: uint32(lowerPort),
			UpperPort: uint32(upperPort),
		},
		SourcePortRange: &acl.AccessLists_Acl_Rule_Match_IpRule_PortRange{
			LowerPort: uint32(0),
			UpperPort: uint32(65535),
		},
	}, nil
}

func getIpRule(parsed map[string]string) (*acl.AccessLists_Acl_Rule_Match_IpRule, error) {

	ip, err := getIp(parsed)
	if err != nil {
		return nil, err
	}

	icmp, err := getIcmp(parsed)
	if err != nil {
		return nil, err
	}

	tcp, err := getTcp(parsed)
	if err != nil {
		return nil, err
	}

	udp, err := getUdp(parsed)
	if err != nil {
		return nil, err
	}

	return &acl.AccessLists_Acl_Rule_Match_IpRule{
		Ip:   ip,
		Icmp: icmp,
		Tcp:  tcp,
		Udp:  udp,
	}, nil
}

func getMatch(parsed map[string]string) (*acl.AccessLists_Acl_Rule_Match, error) {

	iprule, err := getIpRule(parsed)
	if err != nil {
		return nil, err
	}

	return &acl.AccessLists_Acl_Rule_Match{
		IpRule:    iprule,
		MacipRule: nil,
	}, nil
}

func (c *aclConverter) ToDataRequest(rv *rpc.DataRequest, connect bool) (*rpc.DataRequest, error) {
	if c == nil {
		return rv, fmt.Errorf("aclConverter cannot be nil")
	}
	// TODO check if 'c' is complete
	if rv == nil {
		rv = &rpc.DataRequest{}
	}

	rules := []*acl.AccessLists_Acl_Rule{}

	for name, rule := range c.Rules {
		parsed := tools.ParseKVStringToMap(rule, ",", "=")

		action, err := getAction(parsed)
		if err != nil {
			logrus.Errorf("Parsing rule %s failed with %v", rule, err)
			return nil, err
		}

		match, err := getMatch(parsed)
		if err != nil {
			logrus.Errorf("Parsing rule %s failed with %v", rule, err)
			return nil, err
		}

		rules = append(rules, &acl.AccessLists_Acl_Rule{
			RuleName:  name,
			AclAction: action,
			Match:     match,
		})

		rv.AccessLists = append(rv.AccessLists, &acl.AccessLists_Acl{
			AclName: c.Name,
			Rules:   rules,
			Interfaces: &acl.AccessLists_Acl_Interfaces{
				Egress: []string{},
				Ingress: []string{
					c.IngressInterface,
				},
			},
		})
	}

	return rv, nil
}
