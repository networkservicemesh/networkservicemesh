package tests

import (
	"context"

	vpp_interfaces "github.com/ligato/vpp-agent/api/models/vpp/interfaces"

	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

// ConnectionDump - opens a Client connection to another Network Service
type ConnectionDump struct {
	IncomingConnection *networkservice.Connection
	OutgoingConnection *networkservice.Connection
	ConnectionMap      map[string]*vpp_interfaces.Interface
}

// Request implements the request handler
// Consumes from ctx context.Context:
//	   Next
func (cce *ConnectionDump) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	incomingConnection := request.GetConnection()
	if incomingConnection != nil {
		cce.IncomingConnection = proto.Clone(incomingConnection).(*networkservice.Connection)
	}
	cce.OutgoingConnection = endpoint.ClientConnection(ctx)
	cce.ConnectionMap = vppagent.ConnectionMap(ctx)

	var err error
	if endpoint.Next(ctx) != nil {
		incomingConnection, err = endpoint.Next(ctx).Request(ctx, request)
		if err != nil {
			return nil, err
		}
	}

	return incomingConnection, nil
}

// Close implements the close handler
// Consumes from ctx context.Context:
//	   Next
func (cce *ConnectionDump) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	cce.IncomingConnection = connection
	cce.OutgoingConnection = endpoint.ClientConnection(ctx)
	cce.ConnectionMap = vppagent.ConnectionMap(ctx)

	if endpoint.Next(ctx) != nil {
		if _, err := endpoint.Next(ctx).Close(ctx, connection); err != nil {
			return &empty.Empty{}, err
		}
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (cce *ConnectionDump) Name() string {
	return "connection.dump"
}

// NewConnectionDump creates a connection dump object
func NewConnectionDump() *ConnectionDump {
	return &ConnectionDump{}
}
