package mechanism_memif

import (
	"context"
	"fmt"
	"path"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/models/vpp"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type memifServer struct {
	baseDir string
}

func NewServer(baseDir string) networkservice.NetworkServiceServer {
	return &memifServer{baseDir: baseDir}
}

func (m *memifServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	m.appendInterfaceConfig(ctx, conn)
	return conn, nil
}

func (m *memifServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	m.appendInterfaceConfig(ctx, conn)
	return next.Server(ctx).Close(ctx, conn)
}

func (m *memifServer) appendInterfaceConfig(ctx context.Context, conn *connection.Connection) {
	if mechanism := memif.ToMechanism(conn.GetMechanism()); mechanism != nil {
		conf := vppagent.Config(ctx)
		conf.GetVppConfig().Interfaces = append(conf.VppConfig.Interfaces, &vpp.Interface{
			Name:    fmt.Sprintf("server-%s", conn.GetId()),
			Type:    vpp_interfaces.Interface_MEMIF,
			Enabled: true,
			Link: &vpp_interfaces.Interface_Memif{
				Memif: &vpp_interfaces.MemifLink{
					Master:         true,
					SocketFilename: path.Join(m.baseDir, mechanism.GetSocketFilename()),
				},
			},
		})
	}
}
