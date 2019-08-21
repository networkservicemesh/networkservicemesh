package endpoint

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

// ConnectionMutator is function that accepts connection and modify it
type ConnectionMutator func(*connection.Connection) error

// CustomFuncEndpoint is endpoint that apply passed ConnectionMutator to connection that accepts from next endpoint
type CustomFuncEndpoint struct {
	connectionMutator ConnectionMutator
	name              string
}

// Request implements Request method from NetworkServiceServer
// Consumes from ctx context.Context:
//	   Next
func (cf *CustomFuncEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if err := cf.connectionMutator(request.GetConnection()); err != nil {
		Log(ctx).Error(err)
		return nil, err
	}

	if Next(ctx) != nil {
		return Next(ctx).Request(ctx, request)
	}

	Log(ctx).Infof("%v endpoint completed on connection: %v", cf.name, request.GetConnection())
	return request.GetConnection(), nil
}

// Close implements Close method from NetworkServiceServer
// Consumes from ctx context.Context:
//	   Next
func (cf *CustomFuncEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (cf *CustomFuncEndpoint) Name() string {
	return "custom"
}

// NewCustomFuncEndpoint create CustomFuncEndpoint
func NewCustomFuncEndpoint(name string, mutator ConnectionMutator) *CustomFuncEndpoint {
	return &CustomFuncEndpoint{
		name:              name,
		connectionMutator: mutator,
	}
}
