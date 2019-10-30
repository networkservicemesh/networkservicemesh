package converter

import (
	"path"

	vpp_l2 "github.com/ligato/vpp-agent/api/models/vpp/l2"

	"github.com/pkg/errors"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
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
		return rv, errors.New("CrossConnectConverter cannot be nil")
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
	dstName := dstPrefix + c.GetId()

	if src := c.GetLocalSource(); src != nil {
		baseDir := path.Join(c.conversionParameters.BaseDir, src.GetMechanism().GetParameters()[common.Workspace])
		conversionParameters := &ConnectionConversionParameters{
			Name:      srcName,
			Terminate: false,
			Side:      SOURCE,
			BaseDir:   baseDir,
		}
		var err error
		rv, err = NewLocalConnectionConverter(src, conversionParameters).ToDataRequest(rv, connect)
		if err != nil {
			return rv, errors.Wrapf(err, "Error Converting CrossConnect %v", c)
		}
	}

	if dst := c.GetLocalDestination(); dst != nil {
		baseDir := path.Join(c.conversionParameters.BaseDir, dst.GetMechanism().GetParameters()[common.Workspace])
		conversionParameters := &ConnectionConversionParameters{
			Name:      dstName,
			Terminate: false,
			Side:      DESTINATION,
			BaseDir:   baseDir,
		}
		var err error
		rv, err = NewLocalConnectionConverter(dst, conversionParameters).ToDataRequest(rv, connect)
		if err != nil {
			return rv, errors.Wrapf(err, "Error Converting CrossConnect %v", c)
		}
	}

	rv, err := c.MechanismsToDataRequest(rv, connect)
	if err != nil {
		return rv, err
	}

	// For connections mechanisms with xconnect required (For example SRv6 does not require xconnect)
	if len(rv.VppConfig.Interfaces) >= 2 {
		ifaces := rv.VppConfig.Interfaces[len(rv.VppConfig.Interfaces)-2:]
		rv.VppConfig.XconnectPairs = append(rv.VppConfig.XconnectPairs, &vpp_l2.XConnectPair{
			ReceiveInterface:  ifaces[0].Name,
			TransmitInterface: ifaces[1].Name,
		})
		rv.VppConfig.XconnectPairs = append(rv.VppConfig.XconnectPairs, &vpp_l2.XConnectPair{
			ReceiveInterface:  ifaces[1].Name,
			TransmitInterface: ifaces[0].Name,
		})
	}

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
	dstName := dstPrefix + c.GetId()

	var err error
	if src := c.GetRemoteSource(); src != nil {
		rv, err = NewRemoteConnectionConverter(src, srcName, dstName, SOURCE).ToDataRequest(rv, connect)
		if err != nil {
			return rv, errors.Wrapf(err, "error Converting CrossConnect %v", c)
		}
	}

	if dst := c.GetRemoteDestination(); dst != nil {
		rv, err = NewRemoteConnectionConverter(dst, dstName, srcName, DESTINATION).ToDataRequest(rv, connect)
		if err != nil {
			return rv, errors.Wrapf(err, "error Converting CrossConnect %v", c)
		}
	}

	return rv, nil
}
