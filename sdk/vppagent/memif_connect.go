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
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

// MemifConnect is a VPP Agent Memif Connect composite
type MemifConnect struct {
	endpoint.BaseCompositeEndpoint
	Workspace   string
	Connections map[string]*ConnectionData
}

// Request implements the request handler
func (mc *MemifConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if mc.GetNext() == nil {
		err := fmt.Errorf("composite requires that there is Next set")
		return nil, err
	}

	incomingConnection, err := mc.GetNext().Request(ctx, request)
	if err != nil {
		return nil, err
	}

	connectionData, err := getConnectionData(mc.GetNext(), incomingConnection, true)
	if err != nil {
		return nil, err
	}
	if connectionData == nil {
		connectionData = &ConnectionData{}
	}

	socketFilename := path.Join(mc.Workspace, incomingConnection.GetMechanism().GetSocketFilename())
	socketDir := path.Dir(socketFilename)

	if err := os.MkdirAll(socketDir, os.ModePerm); err != nil {
		return nil, err
	}

	name := incomingConnection.GetId()
	connectionData.InConnName = name

	var ipAddresses []string
	dstIPAddr := incomingConnection.GetContext().GetIpContext().GetDstIpAddr()
	if dstIPAddr != "" {
		ipAddresses = []string{dstIPAddr}
	}

	connectionData.DataChange = mc.appendDataChange(connectionData.DataChange, name, ipAddresses, socketFilename)

	mc.Connections[incomingConnection.GetId()] = connectionData

	return incomingConnection, nil
}

// Close implements the close handler
func (mc *MemifConnect) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if mc.GetNext() != nil {
		return mc.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// GetOpaque will return the corresponding connection data
func (mc *MemifConnect) GetOpaque(incoming interface{}) interface{} {
	incomingConnection := incoming.(*connection.Connection)
	if connectionData, ok := mc.Connections[incomingConnection.GetId()]; ok {
		return connectionData
	}
	logrus.Errorf("GetOpaque outgoing not found for %v", incomingConnection)
	return nil
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
		Workspace:   configuration.Workspace,
		Connections: map[string]*ConnectionData{},
	}
}

func (mc *MemifConnect) appendDataChange(rv *configurator.Config, name string, ipAddresses []string, socketFilename string) *configurator.Config {
	if rv == nil {
		rv = &configurator.Config{}
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
				Master:         true,
				SocketFilename: socketFilename,
			},
		},
	})

	return rv
}
