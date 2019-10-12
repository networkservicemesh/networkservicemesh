package vxlan

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/pkg/errors"
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
	return nil, errors.Errorf("Not of Mechanism.Type == %s: %+v", MECHANISM, m)
}

func (m *mechanism) SrcIP() (string, error) {
	return common.GetSrcIP(m.Mechanism)
}

func (m *mechanism) DstIp() (string, error) {
	return common.GetDstIP(m.Mechanism)
}