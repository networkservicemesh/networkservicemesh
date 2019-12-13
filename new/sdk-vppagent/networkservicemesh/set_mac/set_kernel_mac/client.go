package set_vpp_ip

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent"
	"google.golang.org/grpc"
)

type setKernelMacClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &setKernelMacClient{}
}

func (s *setKernelMacClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	conf := vppagent.Config(ctx)
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if mechanism := kernel.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		index := len(conf.GetLinuxConfig().GetInterfaces()) - 1
		conf.GetLinuxConfig().GetInterfaces()[index+1].PhysAddress = conn.GetContext().GetEthernetContext().GetSrcMac()
	}
	return conn, nil
}

func (s *setKernelMacClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conf := vppagent.Config(ctx)
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && len(conf.GetLinuxConfig().GetInterfaces()) > 0 {
		conf.GetLinuxConfig().GetInterfaces()[len(conf.GetVppConfig().GetInterfaces())-1].PhysAddress = conn.GetContext().GetEthernetContext().GetSrcMac()
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
