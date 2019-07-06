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
)

// XConnect is a VPP Agent Cross Connect composite
type XConnect struct {
	Workspace string
}

// Request implements the request handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//	   Next
func (xc *XConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)

	if vppAgentConfig.VppConfig == nil || vppAgentConfig.VppConfig.Interfaces == nil || len(vppAgentConfig.VppConfig.Interfaces) <= 2 {
		return nil, fmt.Errorf("vppAgentConfig lacks 2 interfaces to cross connect")
	}

	if vppAgentConfig.VppConfig.Interfaces[0].Name == "" {
		err := fmt.Errorf("received empty incoming connection name")
		return nil, err
	}
	if vppAgentConfig.VppConfig.Interfaces[1].Name == "" {
		err := fmt.Errorf("received empty outgoing connection name")
		return nil, err
	}

	if err := xc.appendDataChange(vppAgentConfig, vppAgentConfig.VppConfig.Interfaces[0].Name, vppAgentConfig.VppConfig.Interfaces[1].Name); err != nil {
		return nil, err
	}
	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Request(ctx, request)
	}

	return request.GetConnection(), nil
}

// Close implements the close handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//	   Next
func (xc *XConnect) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)

	if vppAgentConfig.VppConfig == nil || vppAgentConfig.VppConfig.Interfaces == nil || len(vppAgentConfig.VppConfig.Interfaces) <= 2 {
		return nil, fmt.Errorf("vppAgentConfig lacks 2 interfaces to cross connect")
	}

	if vppAgentConfig.VppConfig.Interfaces[0].Name == "" {
		err := fmt.Errorf("received empty incoming connection name")
		return nil, err
	}
	if vppAgentConfig.VppConfig.Interfaces[1].Name == "" {
		err := fmt.Errorf("received empty outgoing connection name")
		return nil, err
	}

	if err := xc.appendDataChange(vppAgentConfig, vppAgentConfig.VppConfig.Interfaces[0].Name, vppAgentConfig.VppConfig.Interfaces[1].Name); err != nil {
		return nil, err
	}
	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Close(ctx, connection)
	}

	return &empty.Empty{}, nil
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
		Workspace: configuration.Workspace,
	}
}

func (xc *XConnect) appendDataChange(rv *configurator.Config, in, out string) error {
	if rv == nil {
		return fmt.Errorf("XConnect.appendDataChange cannot be called with rv == nil")
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
	return nil
}
