package endpoint

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

// ConnectionMutator is function that accepts connection and modify it
type ConnectionMutator func(*connection.Connection) error

// CustomFuncEndpoint is endpoint that apply passed ConnectionMutator to connection that accepts from next endpoint
type CustomFuncEndpoint struct {
	BaseCompositeEndpoint
	connectionMutator ConnectionMutator
	name              string
}

// Request implements Request method from NetworkServiceServer
func (cf *CustomFuncEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.NetworkServiceReply, error) {
	if cf.GetNext() == nil {
		err := fmt.Errorf("%v endpoint needs next", cf.name)
		logrus.Error(err)
		return nil, err
	}

	reply, err := cf.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	if err := cf.connectionMutator(reply.GetConnection()); err != nil {
		logrus.Error(err)
		return nil, err
	}

	logrus.Infof("%v endpoint completed on connection: %v", cf.name, reply.GetConnection())
	return reply, nil
}

// Close implements Close method from NetworkServiceServer
func (cf *CustomFuncEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if cf.GetNext() != nil {
		return cf.GetNext().Close(ctx, connection)
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
