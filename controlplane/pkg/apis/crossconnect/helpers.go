package crossconnect

import (
	"fmt"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
)

// NewCrossConnect creates a new cross connect
func NewCrossConnect(id, payload string, src, dst connection.Connection) *CrossConnect {
	c := &CrossConnect{
		Id:      id,
		Payload: payload,
	}

	c.SetSourceConnection(src)
	c.SetDestinationConnection(dst)

	return c
}

// GetSourceConnection returns cross connect source connection
func (c *CrossConnect) GetSourceConnection() connection.Connection {
	if src := c.GetRemoteSource(); src != nil {
		return src
	}

	if src := c.GetLocalSource(); src != nil {
		return src
	}

	return nil
}

// SetSourceConnection sets cross connect source connection
func (c *CrossConnect) SetSourceConnection(src connection.Connection) {
	if src.IsRemote() {
		c.Source = &CrossConnect_RemoteSource{
			RemoteSource: src.(*remote.Connection),
		}
	} else {
		c.Source = &CrossConnect_LocalSource{
			LocalSource: src.(*local.Connection),
		}
	}
}

// GetDestinationConnection returns cross connect destination connection
func (c *CrossConnect) GetDestinationConnection() connection.Connection {
	if dst := c.GetRemoteDestination(); dst != nil {
		return dst
	}

	if dst := c.GetLocalDestination(); dst != nil {
		return dst
	}

	return nil
}

// SetDestinationConnection sets cross connect destination connection
func (c *CrossConnect) SetDestinationConnection(dst connection.Connection) {
	if dst.IsRemote() {
		c.Destination = &CrossConnect_RemoteDestination{
			RemoteDestination: dst.(*remote.Connection),
		}
	} else {
		c.Destination = &CrossConnect_LocalDestination{
			LocalDestination: dst.(*local.Connection),
		}
	}
}

func (c *CrossConnect) IsValid() error {
	if c == nil {
		return fmt.Errorf("CrossConnect cannot be nil")
	}

	if c.GetId() == "" {
		return fmt.Errorf("CrossConnect.Id cannot be empty: %v", c)
	}

	if c.GetSource() == nil {
		return fmt.Errorf("CrossConnect.Source cannot be nil: %v", c)
	}

	if c.GetLocalSource() != nil {
		if err := c.GetLocalSource().IsValid(); err != nil {
			return fmt.Errorf("CrossConnect.Source %v invalid: %s", c, err)
		}
	}

	if c.GetRemoteSource() != nil {
		if err := c.GetRemoteSource().IsValid(); err != nil {
			return fmt.Errorf("CrossConnect.Source %v invalid: %s", c, err)
		}
	}

	if c.GetDestination() == nil {
		return fmt.Errorf("CrossConnect.Destination cannot be nil: %v", c)
	}

	if c.GetLocalDestination() != nil {
		if err := c.GetLocalDestination().IsValid(); err != nil {
			return fmt.Errorf("CrossConnect.Destination %v invalid: %s", c, err)
		}
	}

	if c.GetRemoteDestination() != nil {
		if err := c.GetRemoteDestination().IsValid(); err != nil {
			return fmt.Errorf("CrossConnect.Destination %v invalid: %s", c, err)
		}
	}

	if c.GetPayload() == "" {
		return fmt.Errorf("CrossConnect.Payload cannot be empty: %v", c)
	}

	return nil
}

func (c *CrossConnect) IsComplete() error {
	if err := c.IsValid(); err != nil {
		return err
	}

	if c.GetLocalSource() != nil {
		if err := c.GetLocalSource().IsComplete(); err != nil {
			return fmt.Errorf("CrossConnect.Source %v invalid: %s", c, err)
		}
	}

	if c.GetRemoteSource() != nil {
		if err := c.GetRemoteSource().IsComplete(); err != nil {
			return fmt.Errorf("CrossConnect.Source %v invalid: %s", c, err)
		}
	}

	if c.GetLocalDestination() != nil {
		if err := c.GetLocalDestination().IsComplete(); err != nil {
			return fmt.Errorf("CrossConnect.Destination %v invalid: %s", c, err)
		}
	}

	if c.GetRemoteDestination() != nil {
		if err := c.GetRemoteDestination().IsComplete(); err != nil {
			return fmt.Errorf("CrossConnect.Destination %v invalid: %s", c, err)
		}
	}

	return nil
}
