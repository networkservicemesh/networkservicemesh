package bridge

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	l2 "github.com/ligato/vpp-agent/api/models/vpp/l2"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type bridgeServer struct {
	name string
}

func NewServer(name string) networkservice.NetworkServiceServer {
	return &bridgeServer{name: name}
}

func (b *bridgeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	conf := vppagent.Config(ctx)
	b.insertInterfaceIntoBridge(conf)
	return next.Server(ctx).Request(ctx, request)
}

func (b *bridgeServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	conf := vppagent.Config(ctx)
	b.insertInterfaceIntoBridge(conf)
	return next.Server(ctx).Close(ctx, conn)
}

func (b *bridgeServer) insertInterfaceIntoBridge(conf *configurator.Config) {
	if len(conf.GetVppConfig().GetInterfaces()) > 0 {
		conf.GetVppConfig().BridgeDomains = append(conf.GetVppConfig().BridgeDomains, &l2.BridgeDomain{
			Name:                b.name,
			Flood:               false,
			UnknownUnicastFlood: false,
			Forward:             true,
			Learn:               true,
			ArpTermination:      false,
			Interfaces: []*l2.BridgeDomain_Interface{
				{
					Name:                    conf.GetVppConfig().GetInterfaces()[len(conf.GetVppConfig().GetInterfaces())-1].GetName(),
					BridgedVirtualInterface: false,
				},
			},
		})
	}
}
