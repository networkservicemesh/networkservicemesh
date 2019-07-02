package vppagent

import (
	"context"
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
	"github.com/sirupsen/logrus"
)

// MemifConnect is a VPP Agent Memif Connect composite
type MemifConnect struct {
	endpoint.BaseCompositeEndpoint
	Workspace      string
	ConnectionSide ConnectionSide
	Connections    map[string]*ConnectionData
}

// Request implements the request handler
func (mc *MemifConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if mc.GetNext() == nil {
		logrus.Fatal("The VPP Agent Memif Connect composite requires that there is Next set")
	}

	incomingConnection, err := mc.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	var connectionData *ConnectionData
	opaque := mc.GetNext().GetOpaque(incomingConnection)
	if opaque != nil {
		connectionData = opaque.(*ConnectionData)
		if connectionData.DataChange == nil {
			connectionData.DataChange = &configurator.Config{}
		}
	} else {
		connectionData = &ConnectionData{
			DataChange: &configurator.Config{},
		}
	}

	if connectionData.DataChange.VppConfig == nil {
		connectionData.DataChange.VppConfig = &vpp.ConfigData{}
	}

	socketFilename := path.Join(mc.Workspace, incomingConnection.GetMechanism().GetSocketFilename())
	socketDir := path.Dir(socketFilename)

	if err := os.MkdirAll(socketDir, os.ModePerm); err != nil {
		return nil, err
	}

	var name string
	if mc.ConnectionSide == DESTINATION {
		name = "DST-" + incomingConnection.GetId()
	} else {
		name = "SRC-" + incomingConnection.GetId()
	}

	var ipAddresses []string
	if mc.ConnectionSide == DESTINATION && incomingConnection.GetContext().DstIpAddr != "" {
		ipAddresses = []string{incomingConnection.GetContext().DstIpAddr}
	}

	connectionData.DataChange.VppConfig.Interfaces = append(connectionData.DataChange.VppConfig.Interfaces, &vpp.Interface{
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

	mc.Connections[incomingConnection.GetId()] = connectionData
	if mc.ConnectionSide == DESTINATION {
		mc.Connections[incomingConnection.GetId()].DstName = name
	} else {
		mc.Connections[incomingConnection.GetId()].SrcName = name
	}

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

// NewMemifConnect creates a MemifConnect
func NewMemifConnect(configuration *common.NSConfiguration, side ConnectionSide) *MemifConnect {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	return &MemifConnect{
		Workspace:      configuration.Workspace,
		ConnectionSide: side,
		Connections:    map[string]*ConnectionData{},
	}
}
