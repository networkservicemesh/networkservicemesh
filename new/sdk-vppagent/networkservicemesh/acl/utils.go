package acl

import (
	"net"
	"strconv"
	"strings"

	vppacl "github.com/ligato/vpp-agent/api/models/vpp/acl"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
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

func MapToRules(rules map[string]string) ([]*vppacl.ACL_Rule, error) {
	rv := []*vppacl.ACL_Rule{}
	for _, rule := range rules {
		parsed := tools.ParseKVStringToMap(rule, ",", "=")

		action, err := getAction(parsed)
		if err != nil {
			return nil, errors.Errorf("parsing rule %s failed with %v", rule, err)
		}

		match, err := getMatch(parsed)
		if err != nil {
			return nil, errors.Errorf("parsing rule %s failed with %v", rule, err)
		}

		match.Action = action
		rv = append(rv, match)
	}
	return rv, nil
}

func getAction(parsed map[string]string) (vppacl.ACL_Rule_Action, error) {
	actionName, ok := parsed[action]
	if !ok {
		return vppacl.ACL_Rule_Action(0), errors.New("rule should have 'action' set")
	}
	action, ok := vppacl.ACL_Rule_Action_value[strings.ToUpper(actionName)]
	if !ok {
		return vppacl.ACL_Rule_Action(0), errors.New("rule should have a valid 'action'")
	}
	return vppacl.ACL_Rule_Action(action), nil
}

func getIP(parsed map[string]string) (*vppacl.ACL_Rule_IpRule_Ip, error) {
	dstNet, dstNetOk := parsed[dstNet]
	srcNet, srcNetOk := parsed[srcNet]
	if dstNetOk {
		_, _, err := net.ParseCIDR(dstNet)
		if err != nil {
			return nil, errors.Errorf("dstnet is not a valid CIDR [%v]. Failed with: %v", dstNet, err)
		}
	} else {
		dstNet = ""
	}

	if srcNetOk {
		_, _, err := net.ParseCIDR(srcNet)
		if err != nil {
			return nil, errors.Errorf("srcnet is not a valid CIDR [%v]. Failed with: %v", srcNet, err)
		}
	} else {
		srcNet = ""
	}

	if dstNetOk || srcNetOk {
		return &vppacl.ACL_Rule_IpRule_Ip{
			DestinationNetwork: dstNet,
			SourceNetwork:      srcNet,
		}, nil
	}
	return nil, nil
}

func getICMP(parsed map[string]string) (*vppacl.ACL_Rule_IpRule_Icmp, error) {
	icmpType, ok := parsed[icmpType]
	if !ok {
		return nil, nil
	}
	icmpType8, err := strconv.ParseUint(icmpType, 10, 8)
	if err != nil {
		return nil, errors.Errorf("failed parsing icmptype [%v] with: %v", icmpType, err)
	}
	return &vppacl.ACL_Rule_IpRule_Icmp{
		Icmpv6: false,
		IcmpCodeRange: &vppacl.ACL_Rule_IpRule_Icmp_Range{
			First: uint32(0),
			Last:  uint32(65535),
		},
		IcmpTypeRange: &vppacl.ACL_Rule_IpRule_Icmp_Range{
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
		return 0, true, errors.Errorf("failed parsing %s [%v] with: %v", name, port, err)
	}

	return uint16(port16), true, nil
}

func getTCP(parsed map[string]string) (*vppacl.ACL_Rule_IpRule_Tcp, error) {
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

	return &vppacl.ACL_Rule_IpRule_Tcp{
		DestinationPortRange: &vppacl.ACL_Rule_IpRule_PortRange{
			LowerPort: uint32(lowerPort),
			UpperPort: uint32(upperPort),
		},
		SourcePortRange: &vppacl.ACL_Rule_IpRule_PortRange{
			LowerPort: uint32(0),
			UpperPort: uint32(65535),
		},
		TcpFlagsMask:  0,
		TcpFlagsValue: 0,
	}, nil
}

func getUDP(parsed map[string]string) (*vppacl.ACL_Rule_IpRule_Udp, error) {
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

	return &vppacl.ACL_Rule_IpRule_Udp{
		DestinationPortRange: &vppacl.ACL_Rule_IpRule_PortRange{
			LowerPort: uint32(lowerPort),
			UpperPort: uint32(upperPort),
		},
		SourcePortRange: &vppacl.ACL_Rule_IpRule_PortRange{
			LowerPort: uint32(0),
			UpperPort: uint32(65535),
		},
	}, nil
}

func getIPRule(parsed map[string]string) (*vppacl.ACL_Rule_IpRule, error) {
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

	return &vppacl.ACL_Rule_IpRule{
		Ip:   ip,
		Icmp: icmp,
		Tcp:  tcp,
		Udp:  udp,
	}, nil
}

func getMatch(parsed map[string]string) (*vppacl.ACL_Rule, error) {
	ipRule, err := getIPRule(parsed)
	if err != nil {
		return nil, err
	}

	return &vppacl.ACL_Rule{
		IpRule:    ipRule,
		MacipRule: nil,
	}, nil
}
