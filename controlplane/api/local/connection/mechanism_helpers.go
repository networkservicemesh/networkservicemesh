package connection

import (
	"path"
	"strconv"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/pkg/errors"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"
)

// NewMechanism creates a new mechanism with passed type and description.
func NewMechanism(t MechanismType, name, description string) (*Mechanism, error) {
	inodeNum, err := tools.GetCurrentNS()
	if err != nil {
		return nil, err
	}
	rv := &Mechanism{
		Type: t,
		Parameters: map[string]string{
			InterfaceNameKey:        name,
			InterfaceDescriptionKey: description,
			SocketFilename:          path.Join(name, MemifSocket),
			NetNsInodeKey:           inodeNum,
		},
	}
	err = rv.IsValid()
	if err != nil {
		return nil, err
	}
	return rv, nil
}

// IsRemote returns if mechanism type is remote
func (t MechanismType) IsRemote() bool {
	return false
}

// IsRemote returns if mechanism is remote
func (m *Mechanism) IsRemote() bool {
	return false
}

// Equals returns if mechanism equals given mechanism
func (m *Mechanism) Equals(mechanism connection.Mechanism) bool {
	if other, ok := mechanism.(*Mechanism); ok {
		return proto.Equal(m, other)
	}

	return false
}

// Clone clones mechanism
func (m *Mechanism) Clone() connection.Mechanism {
	return proto.Clone(m).(*Mechanism)
}

// GetMechanismType returns mechanism type
func (m *Mechanism) GetMechanismType() connection.MechanismType {
	return m.Type
}

// SetMechanismType sets mechanism type
func (m *Mechanism) SetMechanismType(mechanismType connection.MechanismType) {
	m.Type = mechanismType.(MechanismType)
}

// SetParameters sets mechanism parameters
func (m *Mechanism) SetParameters(parameters map[string]string) {
	m.Parameters = parameters
}

// IsValid checks if mechanism is valid
func (m *Mechanism) IsValid() error {
	if m == nil {
		return errors.New("mechanism cannot be nil")
	}

	if m.GetParameters() == nil {
		return errors.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	if m.Type == MechanismType_KERNEL_INTERFACE {
		if _, ok := m.Parameters[NetNsInodeKey]; !ok {
			return errors.Errorf("mechanism.Type %s requires mechanism.Parameters[%s] for network namespace", m.GetType(), NetNsInodeKey)
		}

		if _, err := strconv.ParseUint(m.Parameters[NetNsInodeKey], 10, 64); err != nil {
			return errors.Wrapf(err, "mechanism.Parameters[%s] must be an unsigned int, instead was: %s: %v", NetNsInodeKey, m.Parameters[NetNsInodeKey], m)
		}

		iface, ok := m.GetParameters()[InterfaceNameKey]
		if !ok {
			return errors.Errorf("mechanism.Type %s mechanism.Parameters[%s] cannot be empty", m.Type, InterfaceNameKey)
		}
		if len(iface) > LinuxIfMaxLength {
			return errors.Errorf("mechanism.Type %s mechanism.Parameters[%s]: %s may not exceed %d characters", m.Type, InterfaceNameKey, m.GetParameters()[InterfaceNameKey], LinuxIfMaxLength)
		}
	}

	if m.Type == MechanismType_MEM_INTERFACE {
		_, ok := m.GetParameters()[InterfaceNameKey]
		if !ok {
			return errors.Errorf("mechanism.Type %s mechanism.Parameters[%s] cannot be empty", m.Type, InterfaceNameKey)
		}
	}

	// TODO: constraints on other Mechanism types

	return nil
}

// IsMemif - mechanism is memif
func (m *Mechanism) IsMemif() bool {
	if m == nil {
		return false
	}
	return m.GetType() == MechanismType_MEM_INTERFACE
}

// IsKernelInterface - mechanism in kernel
func (m *Mechanism) IsKernelInterface() bool {
	if m == nil {
		return false
	}
	return m.GetType() == MechanismType_KERNEL_INTERFACE
}

// GetSocketFilename returns memif mechanism socket filename
func (m *Mechanism) GetSocketFilename() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[SocketFilename]
}

// GetInterfaceName returns mechanism interface name
func (m *Mechanism) GetInterfaceName() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[InterfaceNameKey]
}

// GetNetNsInode returns inode for connection liveness
func (m *Mechanism) GetNetNsInode() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[NetNsInodeKey]
}

// GetDescription returns mechanism description
func (m *Mechanism) GetDescription() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[InterfaceDescriptionKey]
}

// GetWorkspace returns NSM workspace location
func (m *Mechanism) GetWorkspace() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[Workspace]
}

// NetNsFileName - filename of kernel connection socket
func (m *Mechanism) NetNsFileName() (string, error) {
	if m == nil {
		return "", errors.New("mechanism cannot be nil")
	}
	if m.GetParameters() == nil {
		return "", errors.Errorf("Mechanism.Parameters cannot be nil: %v", m)
	}

	if _, ok := m.Parameters[NetNsInodeKey]; !ok {
		return "", errors.Errorf("Mechanism.Type %s requires Mechanism.Parameters[%s] for network namespace", m.GetType(), NetNsInodeKey)
	}

	inodeNum, err := strconv.ParseUint(m.Parameters[NetNsInodeKey], 10, 64)
	if err != nil {
		return "", errors.Errorf("Mechanism.Parameters[%s] must be an unsigned int, instead was: %s: %v", NetNsInodeKey, m.Parameters[NetNsInodeKey], m)
	}
	filename, err := fs.ResolvePodNsByInode(inodeNum)
	if err != nil {
		return "", errors.Wrapf(err, "no file found in /proc/*/ns/net with inode %d: %v", inodeNum)
	}
	return filename, nil
}
