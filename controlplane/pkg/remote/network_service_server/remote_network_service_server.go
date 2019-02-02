package network_service_server

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote_connection_monitor"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
)

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
)

type remoteNetworkServiceServer struct {
	model           model.Model
	serviceRegistry serviceregistry.ServiceRegistry
	monitor         *remote_connection_monitor.RemoteConnectionMonitor
	manager         nsm.NetworkServiceManager
}

func NewRemoteNetworkServiceServer(model model.Model, manager nsm.NetworkServiceManager, serviceRegistry serviceregistry.ServiceRegistry, connectionMonitor *remote_connection_monitor.RemoteConnectionMonitor) remote_networkservice.NetworkServiceServer {
	server := &remoteNetworkServiceServer{
		model:           model,
		serviceRegistry: serviceRegistry,
		monitor:         connectionMonitor,
		manager:         manager,
	}
	return server
}

func (srv *remoteNetworkServiceServer) Request(ctx context.Context, request *remote_networkservice.NetworkServiceRequest) (*remote_connection.Connection, error) {
	logrus.Infof("RemoteNSMD: Received request from client to connect to NetworkService: %v", request)
	params := map[string]string{}

	conn, err := srv.manager.Request(ctx, request, params)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	result := conn.(*remote_connection.Connection)
	srv.monitor.Update(result)

	logrus.Info("RemoteNSMD: Dataplane configuration done...")
	return result, nil
}

func (srv *remoteNetworkServiceServer) Close(ctx context.Context, connection *remote_connection.Connection) (*empty.Empty, error) {
	logrus.Infof("Remote closing connection: %v", *connection)
	clientConnection := srv.model.GetClientConnection(connection.GetId())
	if clientConnection == nil {
		return nil, fmt.Errorf("There is no such client connection %v", connection)
	}
	err := srv.manager.Close(ctx, clientConnection)
	if err != nil {
		logrus.Errorf("Error during connection close: %v", err)
	}
	srv.monitor.Delete(connection)
	return &empty.Empty{}, nil
}