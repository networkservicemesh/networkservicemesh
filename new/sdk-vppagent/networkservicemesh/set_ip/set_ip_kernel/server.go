package set_ip_kernel

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type setKernelIpServer struct{}

func NewServer() networkservice.NetworkServiceServer {
	return &setKernelIpServer{}
}

func (s *setKernelIpServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	conf := vppagent.Config(ctx)
	conn, err := next.Client(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	if mechanism := kernel.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		index := len(conf.GetLinuxConfig().GetInterfaces()) - 1
		conf.GetLinuxConfig().GetInterfaces()[index+1].IpAddresses = []string{conn.GetContext().GetIpContext().GetDstIpAddr()}
	}
	return conn, nil
}

func (s *setKernelIpServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	conf := vppagent.Config(ctx)
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && len(conf.GetLinuxConfig().GetInterfaces()) > 0 {
		conf.GetLinuxConfig().GetInterfaces()[len(conf.GetLinuxConfig().GetInterfaces())-1].IpAddresses = []string{conn.GetContext().GetIpContext().GetDstIpAddr()}
	}
	return next.Client(ctx).Close(ctx, conn)
}
