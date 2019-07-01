package vpp_agent

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/converter"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
)

type VppAgentMemifConnect struct {
	endpoint.BaseCompositeEndpoint
	Workspace        string
	MemifConnections map[string]*ConnectionData
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

	interfaceName := "DST-" + incomingConnection.GetId()
	conversionParameters := &converter.ConnectionConversionParameters{
		Name:      interfaceName,
		Terminate: true,
		Side:      converter.DESTINATION,
		BaseDir:   vmc.Workspace,
	}

	dataChange, err := converter.NewMemifInterfaceConverter(incomingConnection, conversionParameters).ToDataRequest(nil, true)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	vmc.MemifConnections[incomingConnection.GetId()] = &ConnectionData{
		InterfaceName: interfaceName,
		DataChange:    dataChange,
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

// GetOpaque will return the corresponding Memif connection
func (vmc *VppAgentMemifConnect) GetOpaque(incoming interface{}) interface{} {
	incomingConnection := incoming.(*connection.Connection)
	if memifConnection, ok := vmc.MemifConnections[incomingConnection.GetId()]; ok {
		return memifConnection
	}
	logrus.Errorf("GetOpaque outgoing not found for %v", incomingConnection)
	return nil
}

// NewVppAgentMemifConnect creates a VppAgentMemifConnect
func NewVppAgentMemifConnect(configuration *common.NSConfiguration) *VppAgentMemifConnect {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	return &VppAgentMemifConnect{
		Workspace:        configuration.Workspace,
		MemifConnections: map[string]*ConnectionData{},
	}
}
