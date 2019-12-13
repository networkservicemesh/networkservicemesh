package mechanism_vxlan

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/models/vpp"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"google.golang.org/grpc"
)

type vxlanClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &vxlanClient{}
}

func (v *vxlanClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	v.appendInterfaceConfig(ctx, request.GetConnection())
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (v *vxlanClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	v.appendInterfaceConfig(ctx, conn)
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (v *vxlanClient) appendInterfaceConfig(ctx context.Context, conn *connection.Connection) error {
	conf := vppagent.Config(ctx)
	if mechanism := vxlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		srcIp, err := mechanism.SrcIP()
		if err != nil {
			return err
		}
		dstIp, err := mechanism.DstIP()
		if err != nil {
			return err
		}
		vni, err := mechanism.VNI()
		if err != nil {
			return err
		}
		conf.GetVppConfig().Interfaces = append(conf.GetVppConfig().Interfaces, &vpp.Interface{
			Name:    conn.GetId(),
			Type:    vpp_interfaces.Interface_VXLAN_TUNNEL,
			Enabled: true,
			Link: &vpp_interfaces.Interface_Vxlan{
				Vxlan: &vpp_interfaces.VxlanLink{
					SrcAddress: dstIp,
					DstAddress: srcIp,
					Vni:        vni,
				},
			},
		})
	}
	return nil
}
