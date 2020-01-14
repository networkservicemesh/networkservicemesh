package set_ip_vpp

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type setVppIpClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &setVppIpClient{}
}

func (s *setVppIpClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	conf := vppagent.Config(ctx)
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if mechanism := kernel.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		index := len(conf.GetVppConfig().GetInterfaces()) - 1
		conf.GetVppConfig().GetInterfaces()[index+1].IpAddresses = []string{conn.GetContext().GetIpContext().GetSrcIpAddr()}
	}
	return conn, nil
}

func (s *setVppIpClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conf := vppagent.Config(ctx)
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && len(conf.GetLinuxConfig().GetInterfaces()) > 0 {
		conf.GetVppConfig().GetInterfaces()[len(conf.GetVppConfig().GetInterfaces())-1].IpAddresses = []string{conn.GetContext().GetIpContext().GetSrcIpAddr()}
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
