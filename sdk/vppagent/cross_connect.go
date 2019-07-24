package vppagent

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	l2 "github.com/ligato/vpp-agent/api/models/vpp/l2"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

// XConnect is a VPP Agent Cross Connect composite
type XConnect struct {
	endpoint.BaseCompositeEndpoint
	Workspace   string
	Connections map[string]*ConnectionData
}

// Request implements the request handler
func (xc *XConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if xc.GetNext() == nil {
		err := fmt.Errorf("composite requires that there is Next set")
		return nil, err
	}

	incomingConnection, err := xc.GetNext().Request(ctx, request)
	if err != nil {
		return nil, err
	}

	connectionData, err := getConnectionData(xc.GetNext(), incomingConnection, false)
	if err != nil {
		return nil, err
	}

	if connectionData.InConnName == "" {
		err := fmt.Errorf("received empty incoming connection name")
		return nil, err
	}
	if connectionData.OutConnName == "" {
		err := fmt.Errorf("received empty outgoing connection name")
		return nil, err
	}

	connectionData.DataChange = xc.appendDataChange(connectionData.DataChange, connectionData.InConnName, connectionData.OutConnName)

	xc.Connections[incomingConnection.GetId()] = connectionData
	return incomingConnection, nil
}

// Close implements the close handler
func (xc *XConnect) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if xc.GetNext() != nil {
		return xc.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// GetOpaque will return the corresponding connection data
func (xc *XConnect) GetOpaque(incoming interface{}) interface{} {
	incomingConnection := incoming.(*connection.Connection)
	if connectionData, ok := xc.Connections[incomingConnection.GetId()]; ok {
		return connectionData
	}
	logrus.Errorf("GetOpaque outgoing not found for %v", incomingConnection)
	return nil
}

// Name returns the composite name
func (xc *XConnect) Name() string {
	return "cross-connect"
}

// NewXConnect creates a XConnect
func NewXConnect(configuration *common.NSConfiguration) *XConnect {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	return &XConnect{
		Workspace:   configuration.Workspace,
		Connections: map[string]*ConnectionData{},
	}
}

func (xc *XConnect) appendDataChange(rv *configurator.Config, in, out string) *configurator.Config {
	if rv == nil {
		rv = &configurator.Config{}
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}

	rv.VppConfig.XconnectPairs = append(rv.VppConfig.XconnectPairs,
		&l2.XConnectPair{
			ReceiveInterface:  in,
			TransmitInterface: out,
		},
		&l2.XConnectPair{
			ReceiveInterface:  out,
			TransmitInterface: in,
		},
	)

	return rv
}
