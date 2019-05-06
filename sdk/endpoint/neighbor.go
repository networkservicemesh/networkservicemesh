package endpoint

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/sirupsen/logrus"
	"net"
)

type NeighborEndpoint struct {
	BaseCompositeEndpoint
}

func (ne *NeighborEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if ne.GetNext() == nil {
		err := fmt.Errorf("Neighbor endpoint needs next")
		logrus.Errorf("%v", err)
		return nil, err
	}

	newConnection, err := ne.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	addrs, err := net.Interfaces()
	if err == nil {
		for _, iface := range addrs {
			adrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, a := range adrs {
				addr, _, _ := net.ParseCIDR(a.String())
				if !addr.IsLoopback() {
					newConnection.Context.IpNeighbors = append(newConnection.Context.IpNeighbors,
						&connectioncontext.IpNeighbor{
							Ip:              addr.String(),
							HardwareAddress: iface.HardwareAddr.String(),
						},
					)
				}
			}
		}
	}

	logrus.Infof("Neighbor endpoint completed on connection: %v", newConnection)
	return newConnection, nil
}

func (ne *NeighborEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if ne.GetNext() != nil {
		return ne.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

func NewNeighborEndpoint() *NeighborEndpoint {
	return &NeighborEndpoint{}
}
