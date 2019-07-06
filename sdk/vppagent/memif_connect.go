package vppagent

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

// MemifConnect is a VPP Agent Memif Connect composite
type MemifConnect struct {
	Workspace string
}

// Request implements the request handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//     ConnectionMap
func (mc *MemifConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)
	if err := appendMemifInterface(vppAgentConfig, request.GetConnection(), mc.Workspace, true); err != nil {
		return nil, err
	}

	ctx = WithConnectionMap(ctx) // Guarantees we will retrieve a non-nil Connectionmap from context.Context
	connectionMap := ConnectionMap(ctx)
	interfaces := vppAgentConfig.VppConfig.Interfaces
	connectionMap[request.GetConnection()] = interfaces[len(interfaces)-1]

	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Request(ctx, request)
	}

	return request.GetConnection(), nil
}

// Close implements the close handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//     ConnectionMap
//	   Next
func (mc *MemifConnect) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)
	if err := appendMemifInterface(vppAgentConfig, connection, mc.Workspace, true); err != nil {
		return nil, err
	}

	ctx = WithConnectionMap(ctx) // Guarantees we will retrieve a non-nil Connectionmap from context.Context
	connectionMap := ConnectionMap(ctx)
	interfaces := vppAgentConfig.VppConfig.Interfaces
	connectionMap[connection] = interfaces[len(interfaces)-1]

	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (mc *MemifConnect) Name() string {
	return "memif-connect"
}

// NewMemifConnect creates a MemifConnect
func NewMemifConnect(configuration *common.NSConfiguration) *MemifConnect {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	return &MemifConnect{
		Workspace: configuration.Workspace,
	}
}

func appendMemifInterface(rv *configurator.Config, connection *connection.Connection, workspace string, master bool) error {
	socketFilename := path.Join(workspace, connection.GetMechanism().GetSocketFilename())
	socketDir := path.Dir(socketFilename)

	if err := os.MkdirAll(socketDir, os.ModePerm); err != nil {
		return err
	}

	name := connection.GetId()
	var ipAddresses []string
	if master {
		ipAddresses = append(ipAddresses, connection.GetContext().GetIpContext().DstIpAddr)
	} else {
		ipAddresses = append(ipAddresses, connection.GetContext().GetIpContext().SrcIpAddr)
	}

	if rv == nil {
		return fmt.Errorf("MemifConnect.appendDataChange cannot be called with rv == nil")
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}

	rv.VppConfig.Interfaces = append(rv.VppConfig.Interfaces, &vpp.Interface{
		Name:        name,
		Type:        interfaces.Interface_MEMIF,
		Enabled:     true,
		IpAddresses: ipAddresses,
		Link: &interfaces.Interface_Memif{
			Memif: &interfaces.MemifLink{
				Master:         master,
				SocketFilename: socketFilename,
			},
		},
	})
	return nil
}
