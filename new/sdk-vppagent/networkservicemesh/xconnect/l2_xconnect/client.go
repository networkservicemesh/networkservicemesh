package l2_xconnect

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	vpp_l2 "github.com/ligato/vpp-agent/api/models/vpp/l2"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"google.golang.org/grpc"
)

type l2XconnectClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &l2XconnectClient{}
}

func (l *l2XconnectClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	l.appendInterfaceConfig(ctx, request.GetConnection())
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (l *l2XconnectClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	l.appendInterfaceConfig(ctx, conn)
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (l *l2XconnectClient) appendInterfaceConfig(ctx context.Context, conn *connection.Connection) {
	conf := vppagent.Config(ctx)
	if len(conf.GetVppConfig().GetInterfaces()) >= 2 {
		ifaces := conf.GetVppConfig().GetInterfaces()[len(conf.GetVppConfig().Interfaces)-2:]
		conf.GetVppConfig().XconnectPairs = append(conf.GetVppConfig().XconnectPairs, &vpp_l2.XConnectPair{
			ReceiveInterface:  ifaces[0].Name,
			TransmitInterface: ifaces[1].Name,
		})
		conf.GetVppConfig().XconnectPairs = append(conf.GetVppConfig().XconnectPairs, &vpp_l2.XConnectPair{
			ReceiveInterface:  ifaces[1].Name,
			TransmitInterface: ifaces[0].Name,
		})
	}
}
