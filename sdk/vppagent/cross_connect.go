package vppagent

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	l2 "github.com/ligato/vpp-agent/api/models/vpp/l2"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
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
		logrus.Fatal("The VPP Agent Cross Connect composite requires that there is Next set")
	}

	incomingConnection, err := xc.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	opaque := xc.GetNext().GetOpaque(incomingConnection)
	if opaque == nil {
		err := fmt.Errorf("received empty data from Next")
		logrus.Errorf("Unable to find connection data: %v", err)
		return nil, err
	}
	connectionData := opaque.(*ConnectionData)

	if connectionData.DstName == "" {
		err := fmt.Errorf("found empty destination name")
		logrus.Errorf("Invalid connection data: %v", err)
		return nil, err
	}
	if connectionData.SrcName == "" {
		err := fmt.Errorf("found empty source name")
		logrus.Errorf("Invalid connection data: %v", err)
		return nil, err
	}

	connectionData.DataChange = xc.appendDataChange(connectionData.DataChange, connectionData.SrcName, connectionData.DstName)

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

func (xc *XConnect) appendDataChange(rv *configurator.Config, srcName string, dstName string) *configurator.Config {
	if rv == nil {
		rv = &configurator.Config{}
	}
	if rv.VppConfig == nil {
		rv.VppConfig = &vpp.ConfigData{}
	}

	rv.VppConfig.XconnectPairs = append(rv.VppConfig.XconnectPairs,
		&l2.XConnectPair{
			ReceiveInterface:  srcName,
			TransmitInterface: dstName,
		},
		&l2.XConnectPair{
			ReceiveInterface:  dstName,
			TransmitInterface: srcName,
		},
	)

	return rv
}
