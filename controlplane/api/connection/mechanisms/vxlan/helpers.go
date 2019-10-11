package vxlan

import (
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
)

type Mechanism interface {
	SrcIP() (string, error)
	DstIp() (string, error)
}

type mechanism struct {
	*connection.Mechanism
}

func NewMechanism(m *connection.Mechanism) (Mechanism, error) {
	if m.Type == MECHANISM {
		return &mechanism{
			m,
		}, nil
	}
	return nil, fmt.Errorf("Not of Mechanism.Type == %s: %+v", MECHANISM, m)
}

func (m *mechanism) SrcIP() (string, error) {
	return common.GetSrcIP(m.Mechanism)
}

func (m *mechanism) DstIp() (string, error) {
	return common.GetDstIP(m.Mechanism)
}
