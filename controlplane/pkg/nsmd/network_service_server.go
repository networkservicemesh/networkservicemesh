package nsmd

import (
	"fmt"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/serviceregistry"

	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
)

type networkServiceServer struct {
	model           model.Model
	workspace       *Workspace
	monitor         monitor_connection_server.MonitorConnectionServer
	serviceRegistry serviceregistry.ServiceRegistry
}

func NewNetworkServiceServer(model model.Model, workspace *Workspace, serviceRegistry serviceregistry.ServiceRegistry) networkservice.NetworkServiceServer {
	return &networkServiceServer{
		model:           model,
		workspace:       workspace,
		serviceRegistry: serviceRegistry,
	}
}

func (srv *networkServiceServer) getEndpointFromRegistry(ctx context.Context, requestConnection *connection.Connection) *registry.NSERegistration {
	// Get endpoints
	discoveryClient, err := srv.serviceRegistry.NetworkServiceDiscovery()
	if err != nil {
		logrus.Error(err)
		return nil
	}
	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: requestConnection.GetNetworkService(),
	}
	endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
	if err != nil {
		logrus.Error(err)
		return nil
	}
	endpoint := srv.model.GetSelector().SelectEndpoint(requestConnection, endpointResponse.GetNetworkService(), endpointResponse.GetNetworkServiceEndpoints())
	if endpoint == nil {
		return nil
	}
	return &registry.NSERegistration{
		NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[endpoint.GetNetworkServiceManagerName()],
		NetworkserviceEndpoint: endpoint,
		NetworkService:         endpointResponse.GetNetworkService(),
	}
}

func (srv *networkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Received request from client to connect to NetworkService: %v", request)
	// Make sure its a valid request
	err := request.IsValid()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	// Create a ConnectId for the request.GetConnection()
	request.GetConnection().Id = srv.model.ConnectionId()
	// TODO: Mechanism selection
	request.GetConnection().Mechanism = request.MechanismPreferences[0]
	request.GetConnection().GetMechanism().GetParameters()[connection.Workspace] = srv.workspace.Name()

	// get dataplane
	dp, err := srv.model.SelectDataplane()
	if err != nil {
		return nil, err
	}

	logrus.Infof("Preparing to program dataplane: %v...", dp)

	dataplaneClient, dataplaneConn, err := srv.serviceRegistry.DataplaneConnection(dp)
	if err != nil {
		return nil, err
	}
	if dataplaneConn != nil {
		defer dataplaneConn.Close()
	}

	endpoint := srv.getEndpointFromRegistry(ctx, request.GetConnection())
	if err != nil {
		return nil, err
	}

	var dpApiConnection *crossconnect.CrossConnect
	if srv.model.GetNsm().GetName() == endpoint.GetNetworkServiceManager().GetName() {
		dpApiConnection, err = srv.performLocalNSERequest(ctx, request, endpoint)
	} else {
		dpApiConnection, err = srv.performRemoteNSERequest(ctx, request, endpoint, dp)
	}

	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	logrus.Infof("Sending request to dataplane: %v", dpApiConnection)

	rv, err := dataplaneClient.Request(ctx, dpApiConnection)
	if err != nil {
		logrus.Errorf("Dataplane request failed: %s", err)
		return nil, err
	}
	// TODO - be more cautious here about bad return values from Dataplane
	con := rv.GetSource().(*crossconnect.CrossConnect_LocalSource).LocalSource
	srv.workspace.MonitorConnectionServer().UpdateConnection(con)
	logrus.Info("Dataplane configuration done...")
	return con, nil
}

func (srv *networkServiceServer) createLocalNSERequest(endpoint *registry.NSERegistration, request *networkservice.NetworkServiceRequest) *networkservice.NetworkServiceRequest {
	message := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			// TODO track connection ids
			Id:             srv.model.ConnectionId(),
			NetworkService: endpoint.GetNetworkService().GetName(),
			Context:        request.GetConnection().GetContext(),
			Labels:         nil,
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

func (srv *networkServiceServer) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	srv.workspace.MonitorConnectionServer().DeleteConnection(connection)
	return nil, nil
}

func (srv *networkServiceServer) validateNSEConnection(nseConnection *connection.Connection) error {
	err := nseConnection.IsComplete()
	if err != nil {
		err = fmt.Errorf("NetworkService.Request() failed with error: %s", err)
		logrus.Error(err)
		return err
	}
	err = nseConnection.IsComplete()
	if err != nil {
		err = fmt.Errorf("failure Validating NSE Connection: %s", err)
		return err
	}
	return nil
}
func (srv *networkServiceServer) validateRemoteNSEConnection(nseConnection *remote_connection.Connection) error {
	err := nseConnection.IsComplete()
	if err != nil {
		err = fmt.Errorf("NetworkService.Request() failed with error: %s", err)
		logrus.Error(err)
		return err
	}
	err = nseConnection.IsComplete()
	if err != nil {
		err = fmt.Errorf("failure Validating NSE Connection: %s", err)
		return err
	}
	return nil
}

func (srv *networkServiceServer) performLocalNSERequest(ctx context.Context, request *networkservice.NetworkServiceRequest, endpoint *registry.NSERegistration) (*crossconnect.CrossConnect, error) {
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
	err = request.GetConnection().IsComplete()
	if err != nil {
		err = fmt.Errorf("failure Validating NSE Connection: %s", err)
		return nil, err
	}
	workspace := WorkSpaceRegistry().WorkspaceByEndpoint(endpoint.GetNetworkserviceEndpoint())
	if workspace != nil { // In case of tests this could be empty
		nseConnection.GetMechanism().GetParameters()[connection.Workspace] = workspace.Name()
	}
	dpApiConnection := &crossconnect.CrossConnect{
		Id:      request.GetConnection().GetId(),
		Payload: endpoint.GetNetworkService().GetPayload(),
		Source: &crossconnect.CrossConnect_LocalSource{
			request.GetConnection(),
		},
		Destination: &crossconnect.CrossConnect_LocalDestination{
			nseConnection,
		},
	}
	return dpApiConnection, nil
}

func (srv *networkServiceServer) performRemoteNSERequest(ctx context.Context, request *networkservice.NetworkServiceRequest, endpoint *registry.NSERegistration, dataplane *model.Dataplane) (*crossconnect.CrossConnect, error) {
	client, conn, err := srv.serviceRegistry.RemoteNetworkServiceClient(endpoint.GetNetworkServiceManager())
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	if conn != nil {
		defer conn.Close()
	}

	message := srv.createRemoteNSERequest(endpoint, request, dataplane)
	nseConnection, e := client.Request(ctx, message)
	if e != nil {
		logrus.Infof("Received Error in Response to '%s'.Request(%v): %s", message.GetConnection().GetDestinationNetworkServiceManagerName(), message, e)
		return nil, e
	}

	logrus.Infof("Received Reply to '%s'.Request(%v) %v", message.GetConnection().GetDestinationNetworkServiceManagerName(), message, nseConnection)

	if e != nil {
		logrus.Errorf("error requesting networkservice from %+v with message %#v error: %s", endpoint, message, e)
		return nil, e
	}

	err = srv.validateRemoteNSEConnection(nseConnection)
	if err != nil {
		return nil, err
	}

	request.GetConnection().Context = nseConnection.Context
	err = request.GetConnection().IsComplete()
	if err != nil {
		err = fmt.Errorf("failure Validating NSE Connection: %s", err)
		return nil, err
	}

	dpApiConnection := &crossconnect.CrossConnect{
		Id:      request.GetConnection().GetId(),
		Payload: endpoint.GetNetworkService().GetPayload(),
		Source: &crossconnect.CrossConnect_LocalSource{
			request.GetConnection(),
		},
		Destination: &crossconnect.CrossConnect_RemoteDestination{
			nseConnection,
		},
	}
	return dpApiConnection, nil
}
func (srv *networkServiceServer) createRemoteNSERequest(endpoint *registry.NSERegistration, request *networkservice.NetworkServiceRequest, dataplane *model.Dataplane) *remote_networkservice.NetworkServiceRequest {

	// We need to obtain parameters for remote mechanism
	remoteM := []*remote_connection.Mechanism{}

	for _, mechanism := range dataplane.RemoteMechanisms {
		remoteM = append(remoteM, mechanism)
	}

	message := &remote_networkservice.NetworkServiceRequest{
		Connection: &remote_connection.Connection{
			// TODO track connection ids
			Id:                                   srv.model.ConnectionId(),
			NetworkService:                       request.GetConnection().GetNetworkService(),
			Context:                              request.GetConnection().GetContext(),
			Labels:                               request.GetConnection().GetLabels(),
			DestinationNetworkServiceManagerName: endpoint.GetNetworkServiceManager().GetName(),
			SourceNetworkServiceManagerName:      srv.model.GetNsm().GetName(),
			NetworkServiceEndpointName:           endpoint.GetNetworkserviceEndpoint().GetEndpointName(),
		},
		MechanismPreferences: remoteM,
	}
	return message
}
