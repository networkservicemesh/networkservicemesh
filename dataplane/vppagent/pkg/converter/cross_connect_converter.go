package converter

import (
	"fmt"
	"path"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"

	"github.com/ligato/vpp-agent/plugins/vpp/model/l2"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
)

type CrossConnectConverter struct {
	*crossconnect.CrossConnect
	conversionParameters *CrossConnectConversionParameters
	srcInterfaceName     string
	dstInterfaceName     string
}

func NewCrossConnectConverter(c *crossconnect.CrossConnect, conversionParameters *CrossConnectConversionParameters) *CrossConnectConverter {
	return &CrossConnectConverter{
		CrossConnect:         c,
		conversionParameters: conversionParameters,
		srcInterfaceName:     "SRC-" + c.GetId(),
		dstInterfaceName:     "DST-" + c.GetId(),
	}
}

func (c *CrossConnectConverter) ToDataRequest(connect bool) ([]*rpc.DataRequest, error) {
	if c == nil {
		return nil, fmt.Errorf("CrossConnectConverter cannot be nil")
	}
	if err := c.IsComplete(); err != nil {
		return nil, err
	}

	xconRequest, err := c.convertXcon(connect)
	if err != nil {
		return nil, err
	}

	connRequest, err := c.convertConnections(connect)
	if err != nil {
		return nil, err
	}

	if connect {
		return append(connRequest, xconRequest...), nil
	} else {
		return append(xconRequest, connRequest...), nil
	}

}

func (c *CrossConnectConverter) convertXcon(connect bool) ([]*rpc.DataRequest, error) {
	xconRequest := &rpc.DataRequest{}

	xconRequest.XCons = append(xconRequest.XCons, &l2.XConnectPairs_XConnectPair{
		ReceiveInterface:  c.srcInterfaceName,
		TransmitInterface: c.dstInterfaceName,
	})
	xconRequest.XCons = append(xconRequest.XCons, &l2.XConnectPairs_XConnectPair{
		ReceiveInterface:  c.dstInterfaceName,
		TransmitInterface: c.srcInterfaceName,
	})

	return []*rpc.DataRequest{xconRequest}, nil
}

func (c *CrossConnectConverter) convertConnections(connect bool) ([]*rpc.DataRequest, error) {
	var rv []*rpc.DataRequest

	if c.GetLocalSource() != nil {
		baseDir := path.Join(c.conversionParameters.BaseDir, c.GetLocalSource().GetMechanism().GetWorkspace())
		conversionParameters := &ConnectionConversionParameters{
			Name:      c.srcInterfaceName,
			Terminate: false,
			Side:      SOURCE,
			BaseDir:   baseDir,
		}
		requests, err := NewLocalConnectionConverter(c.GetLocalSource(), conversionParameters).ToDataRequest(connect)
		if err != nil {
			return nil, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
		rv = append(rv, requests...)
	}

	if c.GetRemoteSource() != nil {
		requests, err := NewRemoteConnectionConverter(c.GetRemoteSource(), c.srcInterfaceName, SOURCE).ToDataRequest(connect)
		if err != nil {
			return nil, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
		rv = append(rv, requests...)
	}

	if c.GetLocalDestination() != nil {
		baseDir := path.Join(c.conversionParameters.BaseDir, c.GetLocalDestination().GetMechanism().GetWorkspace())
		conversionParameters := &ConnectionConversionParameters{
			Name:      c.dstInterfaceName,
			Terminate: false,
			Side:      DESTINATION,
			BaseDir:   baseDir,
		}
		requests, err := NewLocalConnectionConverter(c.GetLocalDestination(), conversionParameters).ToDataRequest(connect)
		if err != nil {
			return nil, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
		rv = append(rv, requests...)
	}

	if c.GetRemoteDestination() != nil {
		requests, err := NewRemoteConnectionConverter(c.GetRemoteDestination(), c.dstInterfaceName, DESTINATION).ToDataRequest(connect)
		if err != nil {
			return nil, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
		rv = append(rv, requests...)
	}

	return rv, nil
}
