package nsmd

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/local/monitor_connection_server"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	// nseConnectionTimeout defines a timoute for NSM to succeed connection to NSE (seconds)
	nseConnectionTimeout = 15 * time.Second
)

type networkServiceServer struct {
	model             model.Model
	workspace         *Workspace
	monitor           monitor_connection_server.MonitorConnectionServer
	serviceRegistry   serviceregistry.ServiceRegistry
	excluded_prefixes []string
	model.ModelListenerImpl
}

func NewNetworkServiceServer(model model.Model, workspace *Workspace, serviceRegistry serviceregistry.ServiceRegistry, excluded_prefixes []string) networkservice.NetworkServiceServer {
	rv := &networkServiceServer{
		model:             model,
		workspace:         workspace,
		serviceRegistry:   serviceRegistry,
		excluded_prefixes: excluded_prefixes,
	}
	model.AddListener(rv)
	return rv
}

func (srv *networkServiceServer) getEndpointFromRegistry(ctx context.Context, requestConnection *connection.Connection) (*registry.NSERegistration, error) {
	// Get endpoints
	discoveryClient, err := srv.serviceRegistry.NetworkServiceDiscovery()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: requestConnection.GetNetworkService(),
	}
	endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	endpoint := srv.model.GetSelector().SelectEndpoint(requestConnection, endpointResponse.GetNetworkService(), endpointResponse.GetNetworkServiceEndpoints())
	if endpoint == nil {
		return nil, err
	}
	return &registry.NSERegistration{
		NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[endpoint.GetNetworkServiceManagerName()],
		NetworkserviceEndpoint: endpoint,
		NetworkService:         endpointResponse.GetNetworkService(),
	}, nil
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

	endpoint, err := srv.getEndpointFromRegistry(ctx, request.GetConnection())
	if err != nil {
		return nil, err
	}

	// Update Request with exclude_prefixes

	for _, ep := range srv.excluded_prefixes {
		c := request.GetConnection()
		if c.Context == nil {
			c.Context = &connectioncontext.ConnectionContext{}
		}
		c.Context.ExcludedPrefixes = append(c.Context.ExcludedPrefixes, ep)
	}

	var clientConnection *model.ClientConnection
	if srv.model.GetNsm().GetName() == endpoint.GetNetworkServiceManager().GetName() {
		clientConnection, err = srv.performLocalNSERequest(ctx, request, endpoint, dp)
	} else {
		clientConnection, err = srv.performRemoteNSERequest(ctx, request, endpoint, dp)
	}

	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	logrus.Infof("Sending request to dataplane: %v", clientConnection.Xcon)

	clientConnection.Xcon, err = dataplaneClient.Request(ctx, clientConnection.Xcon)
	if err != nil {
		logrus.Errorf("Dataplane request failed: %s", err)
		return nil, err
	}
	srv.model.AddClientConnection(clientConnection)
	// TODO - be more cautious here about bad return values from Dataplane
	con := clientConnection.Xcon.GetSource().(*crossconnect.CrossConnect_LocalSource).LocalSource
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
			{
				Type:       connection.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{},
			},
			{
				Type:       connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{},
			},
		},
	}
	return message
}

func (srv *networkServiceServer) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	logrus.Infof("Closing connection: %v", *connection)
	clientConnection := srv.model.GetClientConnection(connection.Id)
	if clientConnection == nil {
		logrus.Warnf("No connection with id: %s, nothing to close", connection.Id)
		return &empty.Empty{}, nil
	}

	if err := srv.CloseXconOnEndpoint(ctx, clientConnection); err != nil {
		logrus.Error(err)
	}

	if err := srv.CloseXconOnDataplane(context.Background(), clientConnection); err != nil {
		logrus.Error()
	}

	srv.model.DeleteClientConnection(connection.Id)
	srv.workspace.MonitorConnectionServer().DeleteConnection(connection)

	return &empty.Empty{}, nil
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

func (srv *networkServiceServer) performLocalNSERequest(ctx context.Context, request *networkservice.NetworkServiceRequest, endpoint *registry.NSERegistration, dataplane *model.Dataplane) (*model.ClientConnection, error) {
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

	err = request.GetConnection().UpdateContext(nseConnection.Context)
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
			LocalSource: request.GetConnection(),
		},
		Destination: &crossconnect.CrossConnect_LocalDestination{
			LocalDestination: nseConnection,
		},
	}

	clientConnection := &model.ClientConnection{
		ConnectionId: request.Connection.Id,
		Xcon:         dpApiConnection,
		Endpoint:     endpoint,
		Dataplane:    dataplane,
	}
	return clientConnection, nil
}

func (srv *networkServiceServer) performRemoteNSERequest(ctx context.Context, request *networkservice.NetworkServiceRequest, endpoint *registry.NSERegistration, dataplane *model.Dataplane) (*model.ClientConnection, error) {
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
			LocalSource: request.GetConnection(),
		},
		Destination: &crossconnect.CrossConnect_RemoteDestination{
			RemoteDestination: nseConnection,
		},
	}

	clientConnection := &model.ClientConnection{
		ConnectionId: request.Connection.Id,
		Xcon:         dpApiConnection,
		RemoteNsm:    endpoint.GetNetworkServiceManager(),
		Endpoint:     endpoint,
		Dataplane:    dataplane,
	}
	return clientConnection, nil
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

func (srv *networkServiceServer) ClientConnectionUpdated(clientConnection *model.ClientConnection) {
	switch clientConnection.Xcon.State {
	case crossconnect.CrossConnectState_SRC_DOWN:
		if err := srv.CloseXconOnEndpoint(context.Background(), clientConnection); err != nil {
			logrus.Error(err)
		}
		fallthrough
	case crossconnect.CrossConnectState_DST_DOWN:
		if err := srv.CloseXconOnDataplane(context.Background(), clientConnection); err != nil {
			logrus.Error(err)
		}
		srv.model.DeleteClientConnection(clientConnection.ConnectionId)
		break
	}
}

func (srv *networkServiceServer) CloseXconOnEndpoint(ctx context.Context, clientConnection *model.ClientConnection) error {
	if clientConnection.RemoteNsm != nil {
		remoteClient, conn, err := srv.serviceRegistry.RemoteNetworkServiceClient(clientConnection.RemoteNsm)
		if err != nil {
			logrus.Error(err)
			return err
		}
		if conn != nil {
			defer conn.Close()
		}
		logrus.Info("Remote client successfully created")

		if _, err := remoteClient.Close(ctx, clientConnection.Xcon.GetRemoteDestination()); err != nil {
			logrus.Error(err)
			return err
		}
		logrus.Info("Remote part of cross connection successfully closed")
	} else {
		endpointClient, conn, err := srv.serviceRegistry.EndpointConnection(clientConnection.Endpoint)
		if err != nil {
			logrus.Error(err)
			return err
		}
		if conn != nil {
			defer conn.Close()
		}

		logrus.Info("Closing NSE connection...")
		if _, err := endpointClient.Close(ctx, clientConnection.Xcon.GetLocalDestination()); err != nil {
			logrus.Error(err)
			return err
		}
		logrus.Info("NSE connection successfully closed")
	}

	return nil
}

func (srv *networkServiceServer) CloseXconOnDataplane(ctx context.Context, clientConnection *model.ClientConnection) error {
	logrus.Info("Closing cross connection on dataplane...")
	dataplaneClient, conn, err := srv.serviceRegistry.DataplaneConnection(clientConnection.Dataplane)
	if err != nil {
		logrus.Error(err)
		return err
	}
	if conn != nil {
		defer conn.Close()
	}
	if _, err := dataplaneClient.Close(ctx, clientConnection.Xcon); err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Info("Cross connection successfully closed on dataplane")
	return nil
}
