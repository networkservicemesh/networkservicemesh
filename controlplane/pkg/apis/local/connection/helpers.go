package connection

import (
	fmt "fmt"
	"strconv"
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
		_, ok := m.GetParameters()[InterfaceNameKey]
		if !ok {
			return fmt.Errorf("Mechanism.Type %s Mechanism.Parameters[%s] cannot be empty", m.Type, InterfaceNameKey)
		}
	}
	return nil
}
