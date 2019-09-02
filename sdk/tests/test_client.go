package tests

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"

	"github.com/hashicorp/go-multierror"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

// TestClientEndpoint - opens a Client connection to another Network Service
type TestClientEndpoint struct {
	ioConnMap          map[string]*connection.Connection
	outgoingConnection *connection.Connection
}

// Request implements the request handler
// Consumes from ctx context.Context:
//	   Next
func (cce *TestClientEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ctx = endpoint.WithClientConnection(ctx, cce.outgoingConnection)
	incomingConnection := request.GetConnection()
	var err error
	if endpoint.Next(ctx) != nil {
		incomingConnection, err = endpoint.Next(ctx).Request(ctx, request)
		if err != nil {
			return nil, err
		}
	}

	cce.ioConnMap[request.GetConnection().GetId()] = cce.outgoingConnection
	logrus.Infof("outgoingConnection: %v", cce.outgoingConnection)

	return incomingConnection, nil
}

// Close implements the close handler
// Consumes from ctx context.Context:
//	   Next
func (cce *TestClientEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	var result error

	if outgoingConnection, ok := cce.ioConnMap[connection.GetId()]; ok {
		ctx = endpoint.WithClientConnection(ctx, outgoingConnection)
	}

	// Remove connection from map
	defer delete(cce.ioConnMap, connection.GetId())

	if endpoint.Next(ctx) != nil {
		if _, err := endpoint.Next(ctx).Close(ctx, connection); err != nil {
			return &empty.Empty{}, multierror.Append(result, err)
		}
	}

	return &empty.Empty{}, nil
}

// Name returns the composite name
func (cce *TestClientEndpoint) Name() string {
	return "client"
}

// NewTestClientEndpoint -  creates a test ClientEndpoint
func NewTestClientEndpoint(outgoingConnection *connection.Connection) *TestClientEndpoint {
	self := &TestClientEndpoint{
		ioConnMap:          map[string]*connection.Connection{},
		outgoingConnection: outgoingConnection,
	}

	return self
}
