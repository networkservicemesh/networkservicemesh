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
)

type vxlanServer struct{}

func NewServer() networkservice.NetworkServiceServer {
	return &vxlanServer{}
}

func (v *vxlanServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	v.appendInterfaceConfig(ctx, request.GetConnection())
	return next.Server(ctx).Request(ctx, request)
}

func (v *vxlanServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	v.appendInterfaceConfig(ctx, conn)
	return next.Server(ctx).Close(ctx, conn)
}

func (v *vxlanServer) appendInterfaceConfig(ctx context.Context, conn *connection.Connection) error {
	conf := vppagent.Config(ctx)
	if mechanism := vxlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
		// Note: srcIp and Dst Ip are relative to the *client*, and so on the server side are flipped
		srcIp, err := mechanism.DstIP()
		if err != nil {
			return err
		}
		dstIp, err := mechanism.SrcIP()
		if err != nil {
			return err
		}
		// TODO do VNI selection here
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
