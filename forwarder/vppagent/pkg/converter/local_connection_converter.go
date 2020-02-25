package converter

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
)

type LocalConnectionConverter struct {
	*networkservice.Connection
	name         string
	ipAddressKey string
}

// NewLocalConnectionConverter - creates a local connection converter.
func NewLocalConnectionConverter(c *networkservice.Connection, conversionParameters *ConnectionConversionParameters) Converter {
	if c.GetMechanism().GetType() == kernel.MECHANISM {
		return NewKernelConnectionConverter(c, conversionParameters)
	}
	if c.GetMechanism().GetType() == memif.MECHANISM {
		return NewMemifInterfaceConverter(c, conversionParameters)
	}
	return nil
}
