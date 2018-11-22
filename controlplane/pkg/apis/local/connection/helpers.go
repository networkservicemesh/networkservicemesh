package connection

import (
	fmt "fmt"
	"strconv"

	"github.com/ligato/networkservicemesh/pkg/fs"
)

// IsValid - returns nil if Connection is minimally valid.
func (c *Connection) IsValid() error {
	if c == nil {
		return fmt.Errorf("Connection cannot be nil")
	}
	if c.GetNetworkService() == "" {
		return fmt.Errorf("Connection.NetworkService cannot be empty: %v", c)
	}

	if c.GetMechanism() != nil {
		// TODO constraints on other Mechanism types
		if err := c.GetMechanism().IsValid(); err != nil {
			return fmt.Errorf("Invalid Mechanism in %v: %s", c, err)
		}
	}
	return nil
}

func (c *Connection) IsComplete() error {
	if err := c.IsValid(); err != nil {
		return err
	}

	if c.GetId() == "" {
		return fmt.Errorf("Connection.Id cannot be empty: %v", c)
	}

	if c.GetContext() == nil {
		return fmt.Errorf("Connection.Context cannot be nil: %v", c)
	}

	return nil
}

func (m *Mechanism) IsValid() error {
	if m == nil {
		return fmt.Errorf("Mechanism cannot be nil")
	}
	if m.GetParameters() == nil {
		return fmt.Errorf("Mechanism.Parameters cannot be nil: %v", m)
	}

	if m.Type == MechanismType_KERNEL_INTERFACE {
		if _, ok := m.Parameters[NetNsInodeKey]; !ok {
			return fmt.Errorf("Mechanism.Type %s requires Mechanism.Parameters[%s] for network namespace", m.GetType(), NetNsInodeKey)
		}

		if _, err := strconv.ParseUint(m.Parameters[NetNsInodeKey], 10, 64); err != nil {
			return fmt.Errorf("Mechanism.Parameters[%s] must be an unsigned int, instead was: %s: %v", NetNsInodeKey, m.Parameters[NetNsInodeKey], m)
		}

		iface, ok := m.GetParameters()[InterfaceNameKey]
		if !ok {
			return fmt.Errorf("Mechanism.Type %s Mechanism.Parameters[%s] cannot be empty", m.Type, InterfaceNameKey)
		}
		if len(iface) > LinuxIfMaxLength {
			return fmt.Errorf("Mechanism.Type %s Mechanism.Parameters[%s]: %s may not exceed %d characters", m.Type, InterfaceNameKey, m.GetParameters()[InterfaceNameKey], LinuxIfMaxLength)
		}
	}

	if m.Type == MechanismType_MEM_INTERFACE {
		if _, ok := m.Parameters[Workspace]; !ok {
			return fmt.Errorf("Missing Required LocalMechanism.Parameter[%s]", Workspace)
		}

		if master, ok := m.Parameters[Master]; ok {
			if isMaster, err := strconv.ParseBool(master); err != nil || !isMaster {
				return fmt.Errorf("Mechanism.Type %s if Mechanism.Parameters[%s] is specified, it should be 'true'", m.Type, Master)
			}
		}

		isMaster, err := strconv.ParseBool(m.Parameters[Master])
		if err != nil {
			isMaster = false
		}

		isSlave, err := strconv.ParseBool(m.Parameters[Slave])
		if err != nil {
			isSlave = false
		}

		if isMaster && isSlave {
			return fmt.Errorf("Memif mechanism can't be master and slave at the same time")
		}
	}
	return nil
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
