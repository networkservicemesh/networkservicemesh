package crossconnect

import fmt "fmt"

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
