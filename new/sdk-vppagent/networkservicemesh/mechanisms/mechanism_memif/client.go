package mechanism_memif

import (
	"context"
	"fmt"
	"path"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/models/vpp"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type memifClient struct {
	baseDir string
}

func NewClient(baseDir string) networkservice.NetworkServiceClient {
	return &memifClient{baseDir: baseDir}
}

func (m *memifClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
	mechanism := &connection.Mechanism{
		Cls:        cls.LOCAL,
		Type:       memif.MECHANISM,
		Parameters: make(map[string]string),
	}
	request.MechanismPreferences = append(request.MechanismPreferences, mechanism)
	conn, err := next.Client(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	m.appendInterfaceConfig(ctx, conn)
	return conn, nil
}

func (m *memifClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	m.appendInterfaceConfig(ctx, conn)
	return next.Client(ctx).Close(ctx, conn)
}

func (m *memifClient) appendInterfaceConfig(ctx context.Context, conn *connection.Connection) {
	if mechanism := memif.ToMechanism(conn.GetMechanism()); mechanism != nil {
		conf := vppagent.Config(ctx)
		conf.GetVppConfig().Interfaces = append(conf.VppConfig.Interfaces, &vpp.Interface{
			Name:    fmt.Sprintf("client-%s", conn.GetId()),
			Type:    vpp_interfaces.Interface_MEMIF,
			Enabled: true,
			Link: &vpp_interfaces.Interface_Memif{
				Memif: &vpp_interfaces.MemifLink{
					Master:         false,
					SocketFilename: path.Join(m.baseDir, mechanism.GetSocketFilename()),
				},
			},
		})
	}
}
