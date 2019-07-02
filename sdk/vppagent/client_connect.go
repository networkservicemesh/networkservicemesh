package vppagent

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
	"path"
)

// ClientConnect is a VPP Agent Client Connect composite
type ClientConnect struct {
	endpoint.BaseCompositeEndpoint
	Workspace   string
	Connections map[string]*ConnectionData
}

// Request implements the request handler
func (cc *ClientConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if cc.GetNext() == nil {
		logrus.Fatal("The VPP Agent Client Connect composite requires that there is Next set")
	}

	incomingConnection, err := cc.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	opaque := cc.GetNext().GetOpaque(incomingConnection)
	if opaque == nil {
		err := fmt.Errorf("received empty data from Next")
		logrus.Errorf("Unable to find the outgoing connection: %v", err)
		return nil, err
	}
	outgoingConnection := opaque.(*connection.Connection)
	incomingConnection.Context = outgoingConnection.GetContext()

	interfaceName := "DST-" + outgoingConnection.GetId()

	dataChange := &configurator.Config{
		VppConfig: &vpp.ConfigData{
			Interfaces: []*interfaces.Interface{
				{
					Name:    interfaceName,
					Type:    interfaces.Interface_MEMIF,
					Enabled: true,
					Link: &interfaces.Interface_Memif{
						Memif: &interfaces.MemifLink{
							Master:         false,
							SocketFilename: path.Join(cc.Workspace, outgoingConnection.GetMechanism().GetSocketFilename()),
						},
					},
				},
			},
		},
	}

	cc.Connections[incomingConnection.GetId()] = &ConnectionData{
		DstName:    interfaceName,
		DataChange: dataChange,
	}

	return incomingConnection, nil
}

// Close implements the close handler
func (cc *ClientConnect) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if cc.GetNext() != nil {
		return cc.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// GetOpaque will return the corresponding connection data
func (cc *ClientConnect) GetOpaque(incoming interface{}) interface{} {
	incomingConnection := incoming.(*connection.Connection)
	if connectionData, ok := cc.Connections[incomingConnection.GetId()]; ok {
		return connectionData
	}
	logrus.Errorf("GetOpaque outgoing not found for %v", incomingConnection)
	return nil
}

// NewClientConnect creates a ClientConnect
func NewClientConnect(configuration *common.NSConfiguration) *ClientConnect {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	return &ClientConnect{
		Workspace:   configuration.Workspace,
		Connections: map[string]*ConnectionData{},
	}
}
