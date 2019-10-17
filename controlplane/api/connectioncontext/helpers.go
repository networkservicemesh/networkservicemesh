package connectioncontext

import (
	"net"

	"github.com/pkg/errors"
)

func (c *ConnectionContext) IsEthernetContextEmtpy() bool {
	return c.EthernetContext == nil || (c.EthernetContext.SrcMac == "" && c.EthernetContext.DstMac == "")
}

// IsValid - checks ConnectionContext validation
func (c *ConnectionContext) IsValid() error {
	if c == nil {
		return errors.New("ConnectionContext should not be nil")
	}
	ip := c.GetIpContext()
	for _, route := range append(ip.GetSrcRoutes(), ip.GetDstRoutes()...) {
		if route.GetPrefix() == "" {
			return errors.Errorf("ConnectionContext.Route.Prefix is required and cannot be empty/nil: %v", ip)
		}
		_, _, err := net.ParseCIDR(route.GetPrefix())
		if err != nil {
			return errors.Errorf("ConnectionContext.Route.Prefix should be a valid CIDR address: %v", ip)
		}
	}

	for _, neighbor := range ip.GetIpNeighbors() {
		if neighbor.GetIp() == "" {
			return errors.Errorf("ConnectionContext.IpNeighbors.Ip is required and cannot be empty/nil: %v", ip)
		}
		if neighbor.GetHardwareAddress() == "" {
			return errors.Errorf("ConnectionContext.IpNeighbors.HardwareAddress is required and cannot be empty/nil: %v", ip)
		}
	}
	return nil
}

// MeetsRequirements - checks required context parameters have bin set
func (c *ConnectionContext) MeetsRequirements(original *ConnectionContext) error {
	if c == nil {
		return errors.New("ConnectionContext should not be nil")
	}

	err := c.IsValid()
	if err != nil {
		return err
	}
	if original.GetIpContext().GetDstIpRequired() && c.GetIpContext().GetDstIpAddr() == "" {
		return errors.Errorf("ConnectionContext.DestIp is required and cannot be empty/nil: %v", c)
	}
	if original.GetIpContext().GetSrcIpRequired() && c.GetIpContext().GetSrcIpAddr() == "" {
		return errors.Errorf("ConnectionContext.SrcIp is required cannot be empty/nil: %v", c)
	}

	return nil
}

//Validate - checks DNSConfig and returns error if DNSConfig is not valid
func (c *DNSConfig) Validate() error {
	if c == nil {
		return errors.New(DNSConfigShouldNotBeNil)
	}
	if len(c.DnsServerIps) == 0 {
		return errors.New(DNSServerIpsShouldHaveRecords)
	}
	return nil
}

// IsValid - checks ExtraPrefixRequest validation
func (c *ExtraPrefixRequest) IsValid() error {
	if c == nil {
		return errors.New("ExtraPrefixRequest should not be nil")
	}

	if c.RequiredNumber < 1 {
		return errors.Errorf("ExtraPrefixRequest.RequiredNumber should be positive number >=1: %v", c)
	}
	if c.RequestedNumber < 1 {
		return errors.Errorf("ExtraPrefixRequest.RequestedNumber should be positive number >=1: %v", c)
	}

	if c.RequiredNumber > c.RequestedNumber {
		return errors.Errorf("ExtraPrefixRequest.RequiredNumber should be less or equal to ExtraPrefixRequest.RequestedNumber >=1: %v", c)
	}

	if c.PrefixLen < 1 {
		return errors.Errorf("ExtraPrefixRequest.PrefixLen should be positive number >=1: %v", c)
	}

	// Check protocols
	if c.AddrFamily == nil {
		return errors.Errorf("ExtraPrefixRequest.AfFamily should not be nil: %v", c)
	}

	switch c.AddrFamily.Family {
	case IpFamily_IPV4:
		if c.PrefixLen > 32 {
			return errors.Errorf("ExtraPrefixRequest.PrefixLen should be positive number >=1 and <=32 for IPv4 %v", c)
		}
	case IpFamily_IPV6:
		if c.PrefixLen > 128 {
			return errors.Errorf("ExtraPrefixRequest.PrefixLen should be positive number >=1 and <=32 for IPv6 %v", c)
		}
	}

	return nil
}
