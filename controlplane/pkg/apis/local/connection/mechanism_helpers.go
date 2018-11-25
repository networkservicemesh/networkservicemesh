package connection

import (
	fmt "fmt"
	"path"
	"strconv"

	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/networkservicemesh/utils/fs"
)

func NewMechanism(t MechanismType, name string, description string) (*Mechanism, error) {
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
	filename, err := fs.FindFileInProc(inodeNum, "/ns/net")
	if err != nil {
		return "", fmt.Errorf("No file found in /proc/*/ns/net with inode %d: %v", inodeNum, err)
	}
	return filename, nil
}
