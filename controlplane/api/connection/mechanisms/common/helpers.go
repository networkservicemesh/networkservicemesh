package common

import (
	"fmt"
	"net"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
)

// SrcIP returns the source IP parameter of the Mechanism
func GetSrcIP(m *connection.Mechanism) (string, error) {
	return getIPParameter(m, SrcIP)
}

// DstIP returns the destination IP parameter of the Mechanism
func GetDstIP(m *connection.Mechanism) (string, error) {
	return getIPParameter(m, DstIP)
}

func getIPParameter(m *connection.Mechanism, name string) (string, error) {
	if m == nil {
		return "", fmt.Errorf("mechanism cannot be nil")
	}

	if m.GetParameters() == nil {
		return "", fmt.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	ip, ok := m.Parameters[name]
	if !ok {
		return "", fmt.Errorf("mechanism.Type %s requires mechanism.Parameters[%s] for the VXLAN tunnel", m.GetType(), name)
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("mechanism.Parameters[%s] must be a valid IPv4 or IPv6 address, instead was: %s: %v", name, ip, m)
	}

	return ip, nil
}
