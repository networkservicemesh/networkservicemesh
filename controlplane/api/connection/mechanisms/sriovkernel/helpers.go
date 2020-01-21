// Package sriovkernel - describe sriovkernel mechanism
package sriovkernel

import (
	"strconv"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"
)

// Mechanism - kernel mechanism helper
type Mechanism interface {
	// NetNsFileName - return ns file name
	NetNsFileName() (string, error)
	// GetNetNsInode - return net ns inode
	GetNetNsInode() string
	GetParameters() map[string]string
}

type mechanism struct {
	*connection.Mechanism
}

// ToMechanism - convert unified mechanism to helper
func ToMechanism(m *connection.Mechanism) Mechanism {
	if m.GetType() == MECHANISM {
		return &mechanism{
			m,
		}
	}
	return nil
}

func (m *mechanism) GetParameters() map[string]string {
	if m == nil {
		return nil
	}
	return m.Parameters
}

func (m *mechanism) GetNetNsInode() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[common.NetNsInodeKey]
}

func (m *mechanism) NetNsFileName() (string, error) {
	if m == nil {
		return "", errors.New("mechanism cannot be nil")
	}
	if m.GetParameters() == nil {
		return "", errors.Errorf("Mechanism.Parameters cannot be nil: %v", m)
	}

	if _, ok := m.Parameters[common.NetNsInodeKey]; !ok {
		return "", errors.Errorf("Mechanism.Type %s requires Mechanism.Parameters[%s] for network namespace", m.GetType(), common.NetNsInodeKey)
	}

	inodeNum, err := strconv.ParseUint(m.Parameters[common.NetNsInodeKey], 10, 64)
	if err != nil {
		return "", errors.Errorf("Mechanism.Parameters[%s] must be an unsigned int, instead was: %s: %v", common.NetNsInodeKey, m.Parameters[common.NetNsInodeKey], m)
	}
	filename, err := fs.ResolvePodNsByInode(inodeNum)
	if err != nil {
		return "", errors.Wrapf(err, "no file found in /proc/*/ns/net with inode %d", inodeNum)
	}
	return filename, nil
}
