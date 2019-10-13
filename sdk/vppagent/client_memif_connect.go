package vppagent

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

// ClientMemifConnect is a VPP Agent Client Memif Connect composite
type ClientMemifConnect struct {
	Workspace string
}

// Request implements the request handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//     ClientConnection
//	   Next
func (cmc *ClientMemifConnect) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)

	incomingConnection := request.GetConnection()
	outgoingConnection := endpoint.ClientConnection(ctx)
	if outgoingConnection == nil {
		return nil, errors.New("endpoint.ClientConnection(ctx) - returned nil value")
	}

	// Copy context to incoming, since it should match.
	incomingConnection.Context = outgoingConnection.Context

	// Socket is constructed from outgoing name
	if err := appendMemifInterface(vppAgentConfig, outgoingConnection, cmc.Workspace, false); err != nil {
		return nil, err
	}
	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Request(ctx, request)
	}

	return request.GetConnection(), nil
}

// Close implements the close handler
// Request implements the request handler
// Provides/Consumes from ctx context.Context:
//     VppAgentConfig
//     ClientConnection
//	   Next
func (cmc *ClientMemifConnect) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	ctx = WithConfig(ctx) // Guarantees we will retrieve a non-nil VppAgentConfig from context.Context
	vppAgentConfig := Config(ctx)

	outgoingConnection := endpoint.ClientConnection(ctx)
	if outgoingConnection == nil {
		return nil, errors.Errorf("endpoint.ClientConnection(ctx) - returned nil value")
	}

	if err := appendMemifInterface(vppAgentConfig, outgoingConnection, cmc.Workspace, false); err != nil {
		return nil, err
	}
	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Close(ctx, connection)
	}

	return &empty.Empty{}, nil
}

// Name returns the composite name
func (cmc *ClientMemifConnect) Name() string {
	return "client-memif-connect"
}

// NewClientMemifConnect creates a ClientMemifConnect
func NewClientMemifConnect(configuration *common.NSConfiguration) *ClientMemifConnect {
	return &ClientMemifConnect{
		Workspace: configuration.Workspace,
	}
}
