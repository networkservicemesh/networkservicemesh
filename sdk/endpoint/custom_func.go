package endpoint

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/sirupsen/logrus"
)

type ConnectionMutator func(*connection.Connection) error

type CustomFuncEndpoint struct {
	BaseCompositeEndpoint
	connectionMutator ConnectionMutator
	name              string
}

func (cf *CustomFuncEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if cf.GetNext() == nil {
		err := fmt.Errorf("%v endpoint needs next", cf.name)
		logrus.Error(err)
		return nil, err
	}

	newConnection, err := cf.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	if err := cf.connectionMutator(newConnection); err != nil {
		logrus.Error(err)
		return nil, err
	}

	logrus.Infof("%v endpoint completed on connection: %v", cf.name, newConnection)
	return newConnection, nil
}

func (re *CustomFuncEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if re.GetNext() != nil {
		return re.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

func NewCustomFuncEndpoint(name string, mutator ConnectionMutator) *CustomFuncEndpoint {
	return &CustomFuncEndpoint{
		name:              name,
		connectionMutator: mutator,
	}
}
