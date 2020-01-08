package vppagent

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/ligato/vpp-agent/api/models/vpp"
	l2 "github.com/ligato/vpp-agent/api/models/vpp/l2"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
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

	if err := xc.validateConfig(vppAgentConfig); err != nil {
		return nil, err
	}

	// For connections mechanisms with xconnect required (For example SRv6 does not require xconnect)
	if len(vppAgentConfig.VppConfig.Interfaces) >= 2 {
		if err := xc.appendDataChange(vppAgentConfig, vppAgentConfig.VppConfig.Interfaces[0].Name, vppAgentConfig.VppConfig.Interfaces[1].Name); err != nil {
			return nil, err
		}
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

	if err := xc.validateConfig(vppAgentConfig); err != nil {
		return nil, err
	}

	// For connections mechanisms with xconnect required (For example SRv6 does not require xconnect)
	if len(vppAgentConfig.VppConfig.Interfaces) >= 2 {
		if err := xc.appendDataChange(vppAgentConfig, vppAgentConfig.VppConfig.Interfaces[0].Name, vppAgentConfig.VppConfig.Interfaces[1].Name); err != nil {
			return nil, err
		}
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
	return &XConnect{
		Workspace: configuration.Workspace,
	}
}

func (xc *XConnect) appendDataChange(rv *configurator.Config, in, out string) error {
	if rv == nil {
		return errors.New("XConnect.appendDataChange cannot be called with rv == nil")
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

func (xc *XConnect) validateConfig(vppAgentConfig *configurator.Config) error {
	if vppAgentConfig.VppConfig == nil || vppAgentConfig.VppConfig.Interfaces == nil || len(vppAgentConfig.VppConfig.Interfaces) < 2 {
		return errors.New("vppAgentConfig lacks 2 interfaces to cross connect")
	}

	if vppAgentConfig.VppConfig.Interfaces[0].Name == "" {
		err := errors.New("received empty incoming connection name")
		return err
	}
	if vppAgentConfig.VppConfig.Interfaces[1].Name == "" {
		err := errors.New("received empty outgoing connection name")
		return err
	}
	return nil
}
