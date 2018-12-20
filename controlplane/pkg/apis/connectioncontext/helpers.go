package connectioncontext

import (
	"fmt"
	"net"
)

func (c *ConnectionContext) IsComplete() error {
	if c == nil {
		return fmt.Errorf("ConnectionContext should not be nil...")
	}
	for _, route := range c.GetRoutes() {
		if route.GetPrefix() == "" {
			return fmt.Errorf("ConnectionContext.Route.Prefix is required and cannot be empty/nil: %v", c)
		}
		_, _, err := net.ParseCIDR(route.GetPrefix())
		if err != nil {
			return fmt.Errorf("ConnectionContext.Route.Prefix should be a valid CIDR address: %v", c)
		}
	}

	for _, neightbor := range c.GetIpNeighbors() {
		if neightbor.GetIp() == "" {
			return fmt.Errorf("ConnectionContext.IpNeighbors.Ip is required and cannot be empty/nil: %v", c)
		}
	}
	return nil
}

func (c *ConnectionContext) MeetsRequirements(original *ConnectionContext) error {
	if c == nil {
		return fmt.Errorf("ConnectionContext should not be nil...")
	}

	err := c.IsComplete()
	if err != nil {
		return err
	}
	if original.GetDstIpRequired() && c.GetDstIpAddr() == "" {
		return fmt.Errorf("ConnectionContext.DestIp is required and cannot be empty/nil: %v", c)
	}
	if original.GetSrcIpRequired() && c.GetSrcIpAddr() == "" {
		return fmt.Errorf("ConnectionContext.SrcIp is required cannot be empty/nil: %v", c)
	}

	return nil
}
