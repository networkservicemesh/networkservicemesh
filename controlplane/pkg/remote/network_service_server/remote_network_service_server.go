package network_service_server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/remote/monitor_connection_server"
	"github.com/ligato/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
)

type remoteNetworkServiceServer struct {
	model           model.Model
	serviceRegistry serviceregistry.ServiceRegistry
	monitor         monitor_connection_server.MonitorConnectionServer
}

func (srv *remoteNetworkServiceServer) Request(ctx context.Context, request *remote_networkservice.NetworkServiceRequest) (*remote_connection.Connection, error) {
	logrus.Infof("RemoteNSMD: Received request from client to connect to NetworkService: %v", request)
	// Make sure its a valid request
	err := request.IsValid()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	// get dataplane
	dp, err := srv.model.SelectDataplane()
	if err != nil {
		return nil, err
	}

	// Create a ConnectId for the request.GetConnection()
	request.GetConnection().Id = srv.model.ConnectionId()

	mechanism, err := srv.selectMechanism(request, dp)

	request.GetConnection().Mechanism = mechanism

	// We need to select a dataplane Remote address to be good one.

	logrus.Infof("Selected Remote Mechanism: %+v", request.MechanismPreferences[0])

	// Get endpoints
	endpoint := srv.model.GetEndpoint(request.GetConnection().GetNetworkServiceEndpointName())

	logrus.Infof("RemoteNSMD: Preparing to program dataplane: %v...", dp)

	dataplaneClient, dataplaneConn, err := srv.serviceRegistry.DataplaneConnection(dp)
	if err != nil {
		return nil, err
	}
	if dataplaneConn != nil {
		defer dataplaneConn.Close()
	}

	var dpApiConnection *crossconnect.CrossConnect

	dpApiConnection, err = srv.performLocalNSERequest(ctx, request, endpoint)

	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	logrus.Infof("RemoteNSMD: Sending request to dataplane: %v", dpApiConnection)

	dpCtx, dpCancel := context.WithTimeout(context.Background(), nseConnectionTimeout)
	defer dpCancel()
	rv, err := dataplaneClient.Request(dpCtx, dpApiConnection)
	if err != nil {
		logrus.Errorf("RemoteNSMD: Dataplane request failed: %s", err)
		return nil, err
	}
	// TODO - be more cautious here about bad return values from Dataplane
	con := rv.GetSource().(*crossconnect.CrossConnect_RemoteSource).RemoteSource
	logrus.Infof("Dataplane: Returned connection obj %+v", con)
	srv.monitor.UpdateConnection(con)
	logrus.Info("RemoteNSMD: Dataplane configuration done...")
	return con, nil
}

func findMechanism(MechanismPreferences []*remote_connection.Mechanism, mechanismType remote_connection.MechanismType) *remote_connection.Mechanism {
	for _, m := range MechanismPreferences {
		if m.Type == mechanismType {
			return m
		}
	}
	return nil
}
func (srv *remoteNetworkServiceServer) selectMechanism(request *remote_networkservice.NetworkServiceRequest, dataplane *model.Dataplane) (*remote_connection.Mechanism, error) {
	for _, mechanism := range request.MechanismPreferences {
		dp_mechanism := findMechanism(dataplane.RemoteMechanisms, remote_connection.MechanismType_VXLAN)
		if dp_mechanism == nil {
			continue
		}
		// TODO: Add other mechanisms support
		if mechanism.Type == remote_connection.MechanismType_VXLAN {
			// Update DST IP to be ours
			remoteSrc := mechanism.Parameters[remote_connection.VXLANSrcIP]
			mechanism.Parameters[remote_connection.VXLANSrcIP] = dp_mechanism.Parameters[remote_connection.VXLANSrcIP]
			mechanism.Parameters[remote_connection.VXLANDstIP] = remoteSrc
			mechanism.Parameters[remote_connection.VXLANVNI] = srv.model.Vni()
		}
		return mechanism, nil
	}
	return nil, errors.New(fmt.Sprintf("Failed to select mechanism. No matched mechanisms found..."))
}

func (srv *remoteNetworkServiceServer) createLocalNSERequest(endpoint *registry.NSERegistration, request *remote_networkservice.NetworkServiceRequest) *networkservice.NetworkServiceRequest {
	message := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			// TODO track connection ids
			Id:             srv.model.ConnectionId(),
			NetworkService: request.GetConnection().GetNetworkService(),
			Context:        request.GetConnection().GetContext(),
			Labels:         request.GetConnection().GetLabels(),
		},
		MechanismPreferences: []*connection.Mechanism{
			&connection.Mechanism{
				Type:       connection.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{},
			},
			&connection.Mechanism{
				Type:       connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{},
			},
		},
	}
	return message
}

func (srv *remoteNetworkServiceServer) performLocalNSERequest(ctx context.Context, request *remote_networkservice.NetworkServiceRequest, endpoint *registry.NSERegistration) (*crossconnect.CrossConnect, error) {
	client, nseConn, err := srv.serviceRegistry.EndpointConnection(endpoint)
	if err != nil {
		return nil, err
	}
	if nseConn != nil {
		defer nseConn.Close()
	}

	message := srv.createLocalNSERequest(endpoint, request)

	nseConnection, e := client.Request(ctx, message)

	if e != nil {
		logrus.Errorf("error requesting networkservice from %+v with message %#v error: %s", endpoint, message, e)
		return nil, e
	}

	err = srv.validateNSEConnection(nseConnection)
	if err != nil {
		return nil, err
	}

	request.GetConnection().Context = nseConnection.Context
	// TODO - this is a terrible dirty way to do this, needs cleanup
	nseConnection.GetMechanism().GetParameters()[connection.Workspace] = srv.serviceRegistry.WorkspaceName(endpoint)
	logrus.Infof("Set ")

	err = request.GetConnection().IsComplete()
	if err != nil {
		err = fmt.Errorf("Failure Validating request.GetConnection(): %s %+v", err, request.GetConnection())
		return nil, err
	}

	dpApiConnection := &crossconnect.CrossConnect{
		Id:      request.GetConnection().GetId(),
		Payload: endpoint.GetNetworkService().GetPayload(),
		Source: &crossconnect.CrossConnect_RemoteSource{
			request.GetConnection(),
		},
		Destination: &crossconnect.CrossConnect_LocalDestination{
			nseConnection,
		},
	}
	return dpApiConnection, nil
}

func (srv *remoteNetworkServiceServer) validateNSEConnection(nseConnection *connection.Connection) error {
	err := nseConnection.IsComplete()
	if err != nil {
		err = fmt.Errorf("NetworkService.Request().LocalNSE failed with error: %s %+v", err, nseConnection)
		logrus.Error(err)
		return err
	}
	err = nseConnection.IsComplete()
	if err != nil {
		err = fmt.Errorf("NetworkService.Request().LocalNSE failed validating NSE Connection: %s %+v", err, nseConnection)
		return err
	}
	return nil
}

func (srv *remoteNetworkServiceServer) Close(ctx context.Context, connection *remote_connection.Connection) (*empty.Empty, error) {
	srv.monitor.DeleteConnection(connection)
	//TODO: Add call to dataplane
	return nil, nil
}

func NewRemoteNetworkServiceServer(model model.Model, serviceRegistry serviceregistry.ServiceRegistry, grpcServer *grpc.Server) {
	server := &remoteNetworkServiceServer{
		model:           model,
		serviceRegistry: serviceRegistry,
		monitor:         monitor_connection_server.NewMonitorConnectionServer(),
	}
	remote_networkservice.RegisterNetworkServiceServer(grpcServer, server)
}
