package endpoint

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/sirupsen/logrus"
)

type RouteEndpoint struct {
	BaseCompositeEndpoint
	routes []string
}

func (re *RouteEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if re.GetNext() == nil {
		err := fmt.Errorf("Route endpoint needs next")
		logrus.Errorf("%v", err)
		return nil, err
	}

	newConnection, err := re.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	for _, r := range re.routes {
		newConnection.Context.Routes = append(newConnection.Context.Routes, &connectioncontext.Route{
			Prefix: r,
		})
	}

	logrus.Infof("Route endpoint completed on connection: %v", newConnection)
	return newConnection, nil
}

func (re *RouteEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if re.GetNext() != nil {
		return re.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

func NewRouteEndpoint(configuration *common.NSConfiguration) *RouteEndpoint {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	return &RouteEndpoint{
		routes: configuration.Routes,
	}
}
