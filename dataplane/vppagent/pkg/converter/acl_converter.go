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
)

type aclConverter struct {
	Name             string
	Rules            map[string]string
	IngressInterface string
}

// NewAclConverter creates a new ACL converter
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
	action, ok := acl.AclAction_value[action_name]
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
	icmpCode := uint32(0)
	return &acl.AccessLists_Acl_Rule_Match_IpRule_Icmp{
		Icmpv6: false,
		IcmpCodeRange: &acl.AccessLists_Acl_Rule_Match_IpRule_Icmp_Range{
			First: icmpCode,
			Last:  icmpCode,
		},
		IcmpTypeRange: nil,
	}, nil
}

func getTcp(parsed map[string]string) (*acl.AccessLists_Acl_Rule_Match_IpRule_Tcp, error) {
	LowerPort := uint32(0)
	UpperPort := uint32(0)

	return &acl.AccessLists_Acl_Rule_Match_IpRule_Tcp{
		DestinationPortRange: &acl.AccessLists_Acl_Rule_Match_IpRule_PortRange{
			LowerPort: LowerPort,
			UpperPort: UpperPort,
		},
		SourcePortRange: nil,
		TcpFlagsMask:    0,
		TcpFlagsValue:   0,
	}, nil
}

func getUdp(parsed map[string]string) (*acl.AccessLists_Acl_Rule_Match_IpRule_Udp, error) {
	LowerPort := uint32(0)
	UpperPort := uint32(0)

	return &acl.AccessLists_Acl_Rule_Match_IpRule_Udp{
		DestinationPortRange: &acl.AccessLists_Acl_Rule_Match_IpRule_PortRange{
			LowerPort: LowerPort,
			UpperPort: UpperPort,
		},
		SourcePortRange: nil,
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

func (c *aclConverter) ToDataRequest(rv *rpc.DataRequest) (*rpc.DataRequest, error) {

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
