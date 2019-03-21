package converter

import (
	"fmt"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	"github.com/ligato/vpp-agent/api/models/vpp/l2"
	"path"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
)

type CrossConnectConverter struct {
	*crossconnect.CrossConnect
	conversionParameters *CrossConnectConversionParameters
}

func NewCrossConnectConverter(c *crossconnect.CrossConnect, conversionParameters *CrossConnectConversionParameters) *CrossConnectConverter {
	return &CrossConnectConverter{
		CrossConnect:         c,
		conversionParameters: conversionParameters,
	}
}

func (c *CrossConnectConverter) ToDataRequest(rv *configurator.Config, connect bool) (*configurator.Config, error) {
	if c == nil {
		return rv, fmt.Errorf("CrossConnectConverter cannot be nil")
	}
	if err := c.IsComplete(); err != nil {
		return rv, err
	}
	if rv == nil {
		rv = &configurator.Config{}
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}
	if c.GetLocalSource() != nil {
		baseDir := path.Join(c.conversionParameters.BaseDir, c.GetLocalSource().GetMechanism().GetWorkspace())
		conversionParameters := &ConnectionConversionParameters{
			Name:      "SRC-" + c.GetId(),
			Terminate: false,
			Side:      SOURCE,
			BaseDir:   baseDir,
		}
		rv, err := NewLocalConnectionConverter(c.GetLocalSource(), conversionParameters).ToDataRequest(rv, connect)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	if c.GetRemoteSource() != nil {
		rv, err := NewRemoteConnectionConverter(c.GetRemoteSource(), "SRC-"+c.GetId(), SOURCE).ToDataRequest(rv, connect)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	if c.GetLocalDestination() != nil {
		baseDir := path.Join(c.conversionParameters.BaseDir, c.GetLocalDestination().GetMechanism().GetWorkspace())
		conversionParameters := &ConnectionConversionParameters{
			Name:      "DST-" + c.GetId(),
			Terminate: false,
			Side:      DESTINATION,
			BaseDir:   baseDir,
		}
		rv, err := NewLocalConnectionConverter(c.GetLocalDestination(), conversionParameters).ToDataRequest(rv, connect)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	if c.GetRemoteDestination() != nil {
		rv, err := NewRemoteConnectionConverter(c.GetRemoteDestination(), "DST-"+c.GetId(), DESTINATION).ToDataRequest(rv, connect)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	if len(rv.VppConfig.Interfaces) < 2 {
		return nil, fmt.Errorf("Did not create enough interfaces to cross connect, expected at least 2, got %d", len(rv.VppConfig.Interfaces))
	}
	ifaces := rv.VppConfig.Interfaces[len(rv.VppConfig.Interfaces)-2:]
	rv.VppConfig.XconnectPairs = append(rv.VppConfig.XconnectPairs, &vpp_l2.XConnectPair{
		ReceiveInterface:  ifaces[0].Name,
		TransmitInterface: ifaces[1].Name,
	})
	rv.VppConfig.XconnectPairs = append(rv.VppConfig.XconnectPairs, &vpp_l2.XConnectPair{
		ReceiveInterface:  ifaces[1].Name,
		TransmitInterface: ifaces[0].Name,
	})

	return rv, nil
}
