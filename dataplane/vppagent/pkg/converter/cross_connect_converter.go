package converter

import (
	"fmt"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"

	"github.com/ligato/vpp-agent/plugins/vpp/model/l2"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
)

type CrossConnectConverter struct {
	*crossconnect.CrossConnect
}

func NewCrossConnectConverter(c *crossconnect.CrossConnect) *CrossConnectConverter {
	return &CrossConnectConverter{
		CrossConnect: c,
	}
}

func (c *CrossConnectConverter) ToDataRequest(rv *rpc.DataRequest) (*rpc.DataRequest, error) {
	if c == nil {
		return rv, fmt.Errorf("CrossConnectConverter cannot be nil")
	}
	if err := c.IsComplete(); err != nil {
		return rv, err
	}
	if rv == nil {
		rv = &rpc.DataRequest{}
	}

	if c.GetLocalSource() != nil {
		rv, err := NewLocalConnectionConverter(c.GetLocalSource(), "SRC-"+c.GetId(), connectioncontext.SrcIpKey).ToDataRequest(rv)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	if c.GetRemoteSource() != nil {
		rv, err := NewRemoteConnectionConverter(c.GetRemoteSource(), "SRC-"+c.GetId()).ToDataRequest(rv)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	if c.GetLocalDestination() != nil {
		rv, err := NewLocalConnectionConverter(c.GetLocalDestination(), "DST-"+c.GetId(), connectioncontext.DstIpKey).ToDataRequest(rv)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	if c.GetRemoteDestination() != nil {
		rv, err := NewRemoteConnectionConverter(c.GetRemoteDestination(), "DST-"+c.GetId()).ToDataRequest(rv)
		if err != nil {
			return rv, fmt.Errorf("Error Converting CrossConnect %v: %s", c, err)
		}
	}

	if len(rv.Interfaces) < 2 {
		return nil, fmt.Errorf("Did not create enough interfaces to cross connect, expected at least 2, got %d", len(rv.Interfaces))
	}
	ifaces := rv.Interfaces[len(rv.Interfaces)-2:]
	rv.XCons = append(rv.XCons, &l2.XConnectPairs_XConnectPair{
		ReceiveInterface:  ifaces[0].Name,
		TransmitInterface: ifaces[1].Name,
	})
	rv.XCons = append(rv.XCons, &l2.XConnectPairs_XConnectPair{
		ReceiveInterface:  ifaces[1].Name,
		TransmitInterface: ifaces[0].Name,
	})

	return rv, nil
}
