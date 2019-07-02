package vppagent

import (
	"context"
	"os"
	"path"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
)

type VppAgentMemifConnect struct {
	endpoint.BaseCompositeEndpoint
	Workspace      string
	ConnectionSide ConnectionSide
	Connections    map[string]*ConnectionData
}

func (vmc *VppAgentMemifConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if vmc.GetNext() == nil {
		logrus.Fatal("The VPP Agent Memif Connect composite requires that there is Next set")
	}

	incomingConnection, err := vmc.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	var connectionData *ConnectionData
	opaque := vmc.GetNext().GetOpaque(incomingConnection)
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

	socketFilename := path.Join(vmc.Workspace, incomingConnection.GetMechanism().GetSocketFilename())
	socketDir := path.Dir(socketFilename)

	if err := os.MkdirAll(socketDir, os.ModePerm); err != nil {
		return nil, err
	}

	var name string
	if vmc.ConnectionSide == DESTINATION {
		name = "DST-" + incomingConnection.GetId()
	} else {
		name = "SRC-" + incomingConnection.GetId()
	}

	var ipAddresses []string
	if vmc.ConnectionSide == DESTINATION && incomingConnection.GetContext().DstIpAddr != "" {
		ipAddresses = []string{incomingConnection.GetContext().DstIpAddr}
	}

	connectionData.DataChange.VppConfig.Interfaces = append(connectionData.DataChange.VppConfig.Interfaces, &vpp.Interface{
		Name:        name,
		Type:        vpp_interfaces.Interface_MEMIF,
		Enabled:     true,
		IpAddresses: ipAddresses,
		Link: &vpp_interfaces.Interface_Memif{
			Memif: &vpp_interfaces.MemifLink{
				Master:         true,
				SocketFilename: socketFilename,
			},
		},
	})

	vmc.Connections[incomingConnection.GetId()] = connectionData
	if vmc.ConnectionSide == DESTINATION {
		vmc.Connections[incomingConnection.GetId()].DstName = name
	} else {
		vmc.Connections[incomingConnection.GetId()].SrcName = name
	}

	return incomingConnection, nil
}

// Close implements the close handler
func (vmc *VppAgentMemifConnect) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if vmc.GetNext() != nil {
		return vmc.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// GetOpaque will return the corresponding connection data
func (vmc *VppAgentMemifConnect) GetOpaque(incoming interface{}) interface{} {
	incomingConnection := incoming.(*connection.Connection)
	if connectionData, ok := vmc.Connections[incomingConnection.GetId()]; ok {
		return connectionData
	}
	logrus.Errorf("GetOpaque outgoing not found for %v", incomingConnection)
	return nil
}

// NewVppAgentMemifConnect creates a VppAgentMemifConnect
func NewVppAgentMemifConnect(configuration *common.NSConfiguration, side ConnectionSide) *VppAgentMemifConnect {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	return &VppAgentMemifConnect{
		Workspace:      configuration.Workspace,
		ConnectionSide: side,
		Connections:    map[string]*ConnectionData{},
	}
}
