package nsmd

import (
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"time"
)

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
)

type networkServiceServer struct {
	model           model.Model
	workspace       *Workspace
	serviceRegistry serviceregistry.ServiceRegistry
	manager         nsm.NetworkServiceManager
}

func NewNetworkServiceServer(model model.Model, workspace *Workspace, manager nsm.NetworkServiceManager, serviceRegistry serviceregistry.ServiceRegistry) networkservice.NetworkServiceServer {
	rv := &networkServiceServer{
		model:           model,
		workspace:       workspace,
		serviceRegistry: serviceRegistry,
		manager:         manager,
	}
	return rv
}

func (srv *networkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	// This parameters will go into selected mechanism
	params := map[string]string{}
	params[connection.Workspace] = srv.workspace.Name()

	conn, err := srv.manager.Request(ctx, request, params)
	if err != nil {
		return nil, err
	}
	result := conn.(*connection.Connection)
	srv.workspace.MonitorConnectionServer().Update(result)
	return result, nil
}

func (srv *networkServiceServer) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	logrus.Infof("Closing connection: %v", *connection)
	clientConnection := srv.model.GetClientConnection(connection.GetId())
	if clientConnection == nil {
		return nil, fmt.Errorf("There is no such client connection %v", connection)
	}
	err := srv.manager.Close(ctx, clientConnection)
	if err != nil {
		logrus.Errorf("Error during connection close: %v", err)
	}
	srv.workspace.MonitorConnectionServer().Delete(connection)
	return &empty.Empty{}, nil
}
