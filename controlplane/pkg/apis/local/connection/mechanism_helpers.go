package connection

import (
	"fmt"
	"path"
	"runtime"
	"strconv"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"
)

// NewMechanism creates a new mechanism with passed type and description.
func NewMechanism(t MechanismType, name, description string) (*Mechanism, error) {
	inodeNum, err := tools.GetCurrentNS()
	if err != nil {
		if runtime.GOOS == "darwin" {
			// No Linux namespace in mac, it is used only for testing, just for test use "1"
			inodeNum = "1"
		} else {
			return nil, err
		}
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
		return fmt.Errorf("mechanism cannot be nil")
	}

	if m.GetParameters() == nil {
		return fmt.Errorf("mechanism.Parameters cannot be nil: %v", m)
	}

	if m.Type == MechanismType_KERNEL_INTERFACE {
		if _, ok := m.Parameters[NetNsInodeKey]; !ok {
			return fmt.Errorf("mechanism.Type %s requires mechanism.Parameters[%s] for network namespace", m.GetType(), NetNsInodeKey)
		}

		if _, err := strconv.ParseUint(m.Parameters[NetNsInodeKey], 10, 64); err != nil {
			return fmt.Errorf("mechanism.Parameters[%s] must be an unsigned int, instead was: %s: %v", NetNsInodeKey, m.Parameters[NetNsInodeKey], m)
		}

		iface, ok := m.GetParameters()[InterfaceNameKey]
		if !ok {
			return fmt.Errorf("mechanism.Type %s mechanism.Parameters[%s] cannot be empty", m.Type, InterfaceNameKey)
		}
		if len(iface) > LinuxIfMaxLength {
			return fmt.Errorf("mechanism.Type %s mechanism.Parameters[%s]: %s may not exceed %d characters", m.Type, InterfaceNameKey, m.GetParameters()[InterfaceNameKey], LinuxIfMaxLength)
		}
	}

	if m.Type == MechanismType_MEM_INTERFACE {
		_, ok := m.GetParameters()[InterfaceNameKey]
		if !ok {
			return fmt.Errorf("mechanism.Type %s mechanism.Parameters[%s] cannot be empty", m.Type, InterfaceNameKey)
		}
	}

	// TODO: constraints on other Mechanism types

	return nil
}

func (m *Mechanism) IsMemif() bool {
	if m == nil {
		return false
	}
	return m.GetType() == MechanismType_MEM_INTERFACE
}

func (m *Mechanism) IsKernelInterface() bool {
	if m == nil {
		return false
	}
	return m.GetType() == MechanismType_KERNEL_INTERFACE
}

func (m *Mechanism) GetSocketFilename() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[SocketFilename]
}

func (m *Mechanism) GetInterfaceName() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[InterfaceNameKey]
}

func (m *Mechanism) GetNetNsInode() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[NetNsInodeKey]
}

func (m *Mechanism) GetDescription() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[InterfaceDescriptionKey]
}

func (m *Mechanism) GetWorkspace() string {
	if m == nil || m.GetParameters() == nil {
		return ""
	}
	return m.GetParameters()[Workspace]
}

func (m *Mechanism) NetNsFileName() (string, error) {
	if m == nil {
		return "", fmt.Errorf("Mechanism cannot be nil")
	}
	if m.GetParameters() == nil {
		return "", fmt.Errorf("Mechanism.Parameters cannot be nil: %v", m)
	}

	if _, ok := m.Parameters[NetNsInodeKey]; !ok {
		return "", fmt.Errorf("Mechanism.Type %s requires Mechanism.Parameters[%s] for network namespace", m.GetType(), NetNsInodeKey)
	}

	inodeNum, err := strconv.ParseUint(m.Parameters[NetNsInodeKey], 10, 64)
	if err != nil {
		return "", fmt.Errorf("Mechanism.Parameters[%s] must be an unsigned int, instead was: %s: %v", NetNsInodeKey, m.Parameters[NetNsInodeKey], m)
	}
	filename, err := fs.ResolvePodNsByInode(inodeNum)
	if err != nil {
		return "", fmt.Errorf("No file found in /proc/*/ns/net with inode %d: %v", inodeNum, err)
	}
	return filename, nil
}
