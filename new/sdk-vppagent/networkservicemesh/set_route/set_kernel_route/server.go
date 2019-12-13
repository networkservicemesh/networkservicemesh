package set_kernel_route

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/models/linux"
	linux_l3 "github.com/ligato/vpp-agent/api/models/linux/l3"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type setKernelRoute struct{}

func NewServer() networkservice.NetworkServiceServer {
	return &setKernelRoute{}
}

func (s *setKernelRoute) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	s.addRoutes(ctx, request.GetConnection())
	return next.Server(ctx).Request(ctx, request)
}

func (s *setKernelRoute) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	s.addRoutes(ctx, conn)
	return next.Server(ctx).Close(ctx, conn)
}

func (s *setKernelRoute) addRoutes(ctx context.Context, conn *connection.Connection) {
	duplicatedPrefixes := make(map[string]bool)
	for _, route := range conn.GetContext().GetIpContext().GetSrcRoutes() {
		if _, ok := duplicatedPrefixes[route.Prefix]; !ok {
			duplicatedPrefixes[route.Prefix] = true
			vppagent.Config(ctx).GetLinuxConfig().Routes = append(vppagent.Config(ctx).GetLinuxConfig().Routes, &linux.Route{
				DstNetwork:        route.Prefix,
				OutgoingInterface: vppagent.Config(ctx).GetLinuxConfig().GetInterfaces()[0].GetName(),
				Scope:             linux_l3.Route_GLOBAL,
				GwAddr:            extractCleanIPAddress(conn.GetContext().GetIpContext().GetDstIpAddr()),
			})
		}
	}
}
