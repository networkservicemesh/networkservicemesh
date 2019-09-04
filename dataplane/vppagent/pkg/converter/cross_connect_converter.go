package converter

import (
	"fmt"
	"path"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	vpp_l2 "github.com/ligato/vpp-agent/api/models/vpp/l2"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
)

const (
	srcPrefix = "SRC-"
	dstPrefix = "DST-"
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

	srcName := srcPrefix + c.GetId()

	if c.GetLocalSource() != nil {
		baseDir := path.Join(c.conversionParameters.BaseDir, c.GetLocalSource().GetMechanism().GetWorkspace())
		conversionParameters := &ConnectionConversionParameters{
			Name:      srcName,
			Terminate: false,
			Side:      SOURCE,
			BaseDir:   baseDir,
		}
		rv, err := NewLocalConnectionConverter(c.GetLocalSource(), conversionParameters).ToDataRequest(rv, connect)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	rv, err := c.MechanismsToDataRequest(rv, connect)
	if err != nil {
		return rv, err
	}

	if len(rv.VppConfig.Interfaces) < 2 {
		return nil, fmt.Errorf("did not create enough interfaces to cross connect, expected at least 2, got %d", len(rv.VppConfig.Interfaces))
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

// MechanismsToDataRequest prepares data change with mechanisms parameters for vppagent
func (c *CrossConnectConverter) MechanismsToDataRequest(rv *configurator.Config, connect bool) (*configurator.Config, error) {
	if rv == nil {
		rv = &configurator.Config{}
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}

	srcName := srcPrefix + c.GetId()

	var err error
	if c.GetLocalDestination() != nil {
		baseDir := path.Join(c.conversionParameters.BaseDir, c.GetLocalDestination().GetMechanism().GetWorkspace())
		conversionParameters := &ConnectionConversionParameters{
			Name:      dstName,
			Terminate: false,
			Side:      DESTINATION,
			BaseDir:   baseDir,
		}
		rv, err = NewLocalConnectionConverter(c.GetLocalDestination(), conversionParameters).ToDataRequest(rv, connect)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	if c.GetRemoteSource() != nil {
		rv, err = NewRemoteConnectionConverter(c.GetRemoteSource(), srcName, SOURCE).ToDataRequest(rv, connect)
		if err != nil {
			return rv, fmt.Errorf("error Converting CrossConnect %v: %s", c, err)
		}
	}

	if c.GetRemoteDestination() != nil {
		rv, err = NewRemoteConnectionConverter(c.GetRemoteDestination(), "DST-"+c.GetId(), DESTINATION).ToDataRequest(rv, connect)
		if err != nil {
			return rv, fmt.Errorf("error Converting CrossConnect %v: %s", c, err)
		}
	}

	return rv, nil
}
