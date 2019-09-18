package nsmd

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
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
	logrus.Infof("Received request from client to connect to NetworkService: %v", request)
	srv.updateMechanisms(request)

	conn, err := srv.manager.Request(ctx, request)
	if err != nil {
		return nil, err
	}
	result := conn.(*connection.Connection)
	srv.workspace.MonitorConnectionServer().Update(result)
	return result, nil
}

func (srv *networkServiceServer) updateMechanisms(request *networkservice.NetworkServiceRequest) {
	// Update passed local mechanism parameters to contains a workspace name
	for _, mechanism := range request.MechanismPreferences {
		if mechanism.Parameters == nil {
			mechanism.Parameters = map[string]string{}
		}
		mechanism.Parameters[connection.Workspace] = srv.workspace.Name()
	}
}

func (srv *networkServiceServer) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	logrus.Infof("Closing connection: %v", *connection)
	// TODO: check carefully  id of closing connection (need dst connection id)
	clientConnection := srv.model.GetClientConnection(connection.GetId())
	if clientConnection == nil {
		err := fmt.Errorf("there is no such client connection %v", connection)
		logrus.Error(err)
		return nil, err
	}
	err := srv.manager.Close(ctx, clientConnection)
	if err != nil {
		logrus.Errorf("Error during connection close: %v", err)
	}
	srv.workspace.MonitorConnectionServer().Delete(connection)
	return &empty.Empty{}, nil
}
