package vppagent

import (
	"context"
	"fmt"
	"path"
	"reflect"

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

// ClientMemifConnect is a VPP Agent Client Memif Connect composite
type ClientMemifConnect struct {
	endpoint.BaseCompositeEndpoint
	Workspace   string
	Connections map[string]*ConnectionData
}

// Request implements the request handler
func (cmc *ClientMemifConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if cmc.GetNext() == nil {
		err := fmt.Errorf("composite requires that there is Next set")
		return nil, err
	}

	incomingConnection, err := cmc.GetNext().Request(ctx, request)
	if err != nil {
		return nil, err
	}

	opaque := cmc.GetNext().GetOpaque(incomingConnection)
	if opaque == nil {
		err = fmt.Errorf("received empty opaque data from Next")
		return nil, err
	}

	outgoingConnection, ok := opaque.(*connection.Connection)
	if !ok {
		err := fmt.Errorf("unexpected opaque data type: expected *connection.Connection, received %v", reflect.TypeOf(opaque))
		return nil, err
	}

	incomingConnection.Context = outgoingConnection.GetContext()

	name := outgoingConnection.GetId()
	socketFileName := path.Join(cmc.Workspace, outgoingConnection.GetMechanism().GetSocketFilename())

	dataChange := cmc.createDataChange(name, socketFileName)

	cmc.Connections[incomingConnection.GetId()] = &ConnectionData{
		OutConnName: name,
		DataChange:  dataChange,
	}

	return incomingConnection, nil
}

// Close implements the close handler
func (cmc *ClientMemifConnect) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if cmc.GetNext() != nil {
		return cmc.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// GetOpaque will return the corresponding connection data
func (cmc *ClientMemifConnect) GetOpaque(incoming interface{}) interface{} {
	incomingConnection := incoming.(*connection.Connection)
	if connectionData, ok := cmc.Connections[incomingConnection.GetId()]; ok {
		return connectionData
	}
	logrus.Errorf("GetOpaque outgoing not found for %v", incomingConnection)
	return nil
}

// Name returns the composite name
func (cmc *ClientMemifConnect) Name() string {
	return "client-memif-connect"
}

// NewClientMemifConnect creates a ClientMemifConnect
func NewClientMemifConnect(configuration *common.NSConfiguration) *ClientMemifConnect {
	// ensure the env variables are processed
	configuration = common.NewNSConfiguration(configuration)

	return &ClientMemifConnect{
		Workspace:   configuration.Workspace,
		Connections: map[string]*ConnectionData{},
	}
}

func (cmc *ClientMemifConnect) createDataChange(interfaceName, socketFileName string) *configurator.Config {
	return &configurator.Config{
		VppConfig: &vpp.ConfigData{
			Interfaces: []*interfaces.Interface{
				{
					Name:    interfaceName,
					Type:    interfaces.Interface_MEMIF,
					Enabled: true,
					Link: &interfaces.Interface_Memif{
						Memif: &interfaces.MemifLink{
							Master:         false,
							SocketFilename: socketFileName,
						},
					},
				},
			},
		},
	}
}
