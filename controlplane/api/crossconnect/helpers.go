package crossconnect

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
)

// NewCrossConnect creates a new crossConnect
func NewCrossConnect(id, payload string, src, dst *networkservice.Connection) *CrossConnect {
	c := &CrossConnect{
		Id:          id,
		Payload:     payload,
		Source:      src,
		Destination: dst,
	}
	return c
}

// IsValid checks if crossConnect is minimally valid
func (c *CrossConnect) IsValid() error {
	if c == nil {
		return errors.New("crossConnect cannot be nil")
	}

	if c.GetId() == "" {
		return errors.Errorf("crossConnect.Id cannot be empty: %v", c)
	}

	src := c.GetSource()
	if src == nil {
		return errors.Errorf("crossConnect.Source cannot be nil: %v", c)
	}

	if err := src.IsValid(); err != nil {
		return errors.Wrapf(err, "crossConnect.Source %v invalid", c)
	}

	dst := c.GetDestination()
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

	if err := c.GetSource().IsComplete(); err != nil {
		return errors.Wrapf(err, "crossConnect.Source %v invalid", c)
	}

	if err := c.GetDestination().IsComplete(); err != nil {
		return errors.Wrapf(err, "crossConnect.Destination %v invalid", c)
	}

	return nil
}

// GetLocalSource - return a source and check if it is local
func (c *CrossConnect) GetLocalSource() *networkservice.Connection {
	if c == nil {
		return nil
	}
	if c.Source.IsRemote() {
		return nil
	}
	return c.Source
}

// GetRemoteSource - return a source and check if it is remote
func (c *CrossConnect) GetRemoteSource() *networkservice.Connection {
	if c == nil {
		return nil
	}
	if !c.Source.IsRemote() {
		return nil
	}
	return c.Source
}

// GetLocalDestination - return a destination and check if it is local
func (c *CrossConnect) GetLocalDestination() *networkservice.Connection {
	if c == nil {
		return nil
	}
	if c.Destination.IsRemote() {
		return nil
	}
	return c.Destination
}

// GetRemoteDestination - return a destination and check if it is remote
func (c *CrossConnect) GetRemoteDestination() *networkservice.Connection {
	if c == nil {
		return nil
	}
	if !c.Destination.IsRemote() {
		return nil
	}
	return c.Destination
}
