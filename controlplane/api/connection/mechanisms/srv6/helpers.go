package srv6

import (
	"net"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
)

// Mechanism - a vxlan mechanism utility wrapper
type Mechanism interface {
	// SrcHostIP -  src localsid of mgmt interface
	SrcHostIP() (string, error)
	// DstHostIP - dst localsid of mgmt interface
	DstHostIP() (string, error)
	// SrcBSID -  src BSID
	SrcBSID() (string, error)
	// DstBSID - dst BSID
	DstBSID() (string, error)
	// SrcLocalSID -  src LocalSID
	SrcLocalSID() (string, error)
	// DstLocalSID - dst LocalSID
	DstLocalSID() (string, error)
	// SrcHardwareAddress -  src hw address
	SrcHardwareAddress() (string, error)
	// DstHardwareAddress - dst hw address
	DstHardwareAddress() (string, error)
}

type mechanism struct {
	*connection.Mechanism
}

func (m mechanism) SrcHostIP() (string, error) {
	return getIPParameter(m.Mechanism, SrcHostIP)
}

func (m mechanism) DstHostIP() (string, error) {
	return getIPParameter(m.Mechanism, DstHostIP)
}

func (m mechanism) SrcBSID() (string, error) {
	return getIPParameter(m.Mechanism, SrcBSID)
}

func (m mechanism) DstBSID() (string, error) {
	return getIPParameter(m.Mechanism, DstBSID)
}

func (m mechanism) SrcLocalSID() (string, error) {
	return getIPParameter(m.Mechanism, SrcLocalSID)
}

func (m mechanism) DstLocalSID() (string, error) {
	return getIPParameter(m.Mechanism, DstLocalSID)
}

func (m mechanism) SrcHardwareAddress() (string, error) {
	return getStringParameter(m.Mechanism, SrcHardwareAddress)
}

func (m mechanism) DstHardwareAddress() (string, error) {
	return getStringParameter(m.Mechanism, DstHardwareAddress)
}

// ToMechanism - convert unified mechanism to useful wrapper
func ToMechanism(m *connection.Mechanism) Mechanism {
	if m.Type == MECHANISM {
		return &mechanism{
			m,
		}
	}
	return nil
}

func getIPParameter(m *connection.Mechanism, name string) (string, error) {
	ip, err := getStringParameter(m, name)
	if err != nil {
		return "", err
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", errors.Errorf("mechanism.Parameters[%s] must be a valid IPv4 or IPv6 address, instead was: %s: %v", name, ip, m)
	}

	return ip, nil
}

func getStringParameter(m *connection.Mechanism, name string) (string, error) {
	if m == nil {
		return "", errors.New("mechanism cannot be nil")
	}

	if m.GetParameters() == nil {
		return "", errors.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	v, ok := m.Parameters[name]
	if !ok {
		return "", errors.Errorf("mechanism.Type %s requires mechanism.Parameters[%s]", m.GetType(), name)
	}

	return v, nil
}
