package crossconnect

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	connection2 "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	connection3 "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/pkg/errors"
)

// NewCrossConnect creates a new crossConnect
func NewCrossConnect(id, payload string, src, dst connection2.Connection) *CrossConnect {
	c := &CrossConnect{
		Id:      id,
		Payload: payload,
	}

	c.SetSourceConnection(src)
	c.SetDestinationConnection(dst)

	return c
}

// GetSourceConnection returns crossConnect source connection
func (c *CrossConnect) GetSourceConnection() connection2.Connection {
	if src := c.GetRemoteSource(); src != nil {
		return src
	}

	if src := c.GetLocalSource(); src != nil {
		return src
	}

	return nil
}

// SetSourceConnection sets crossConnect source connection
func (c *CrossConnect) SetSourceConnection(src connection2.Connection) {
	if src.IsRemote() {
		c.Source = &CrossConnect_RemoteSource{
			RemoteSource: src.(*connection3.Connection),
		}
	} else {
		c.Source = &CrossConnect_LocalSource{
			LocalSource: src.(*connection.Connection),
		}
	}
}

// GetDestinationConnection returns crossConnect destination connection
func (c *CrossConnect) GetDestinationConnection() connection2.Connection {
	if dst := c.GetRemoteDestination(); dst != nil {
		return dst
	}

	if dst := c.GetLocalDestination(); dst != nil {
		return dst
	}

	return nil
}

// SetDestinationConnection sets crossConnect destination connection
func (c *CrossConnect) SetDestinationConnection(dst connection2.Connection) {
	if dst.IsRemote() {
		c.Destination = &CrossConnect_RemoteDestination{
			RemoteDestination: dst.(*connection3.Connection),
		}
	} else {
		c.Destination = &CrossConnect_LocalDestination{
			LocalDestination: dst.(*connection.Connection),
		}
	}
}

// IsValid checks if crossConnect is minimally valid
func (c *CrossConnect) IsValid() error {
	if c == nil {
		return errors.New("crossConnect cannot be nil")
	}

	if c.GetId() == "" {
		return errors.Errorf("crossConnect.Id cannot be empty: %v", c)
	}

	src := c.GetSourceConnection()
	if src == nil {
		return errors.Errorf("crossConnect.Source cannot be nil: %v", c)
	}

	if err := src.IsValid(); err != nil {
		return errors.Wrapf(err, "crossConnect.Source %v invalid", c)
	}

	dst := c.GetDestinationConnection()
	if dst == nil {
		return errors.Errorf("crossConnect.Destination cannot be nil: %v", c)
	}

	if err := dst.IsValid(); err != nil {
		return errors.Wrapf(err, "crossConnect.Destination %v invalid", c)
	}

	if c.GetPayload() == "" {
		return errors.Errorf("crossConnect.Payload cannot be empty: %v", c)
	}

	return nil
}

// IsComplete checks if crossConnect is complete valid
func (c *CrossConnect) IsComplete() error {
	if err := c.IsValid(); err != nil {
		return err
	}

	if err := c.GetSourceConnection().IsComplete(); err != nil {
		return errors.Wrapf(err, "crossConnect.Source %v invalid", c)
	}

	if err := c.GetDestinationConnection().IsComplete(); err != nil {
		return errors.Wrapf(err, "crossConnect.Destination %v invalid", c)
	}

	return nil
}
