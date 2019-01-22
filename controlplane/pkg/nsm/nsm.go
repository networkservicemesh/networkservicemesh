// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package nsm

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

///// Network service manager to manage both local/remote NSE connections.
type networkServiceManager struct {
	serviceRegistry   serviceregistry.ServiceRegistry
	model             model.Model
	excluded_prefixes []string
}

func NewNetworkServiceManager(model model.Model, serviceRegistry serviceregistry.ServiceRegistry, excluded_prefixes []string) nsm.NetworkServiceManager {
	return &networkServiceManager{
		serviceRegistry:   serviceRegistry,
		model:             model,
		excluded_prefixes: excluded_prefixes,
	}
}

func (srv *networkServiceManager) Request(ctx context.Context, request nsm.NSMRequest, extra_parameters map[string]string) (nsm.NSMConnection, error) {
	// Make sure its a valid request
	err := request.IsValid()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	var nsmConnection nsm.NSMConnection = nil
	nsmConnection = srv.cloneConnection(request, nsmConnection)

	// Create a ConnectId for the request.GetConnection(), since connections are managed per NSM
	nsmConnection.SetId(srv.createConnectionId())

	// get dataplane
	dp, err := srv.model.SelectDataplane()
	if err != nil {
		return nil, err
	}

	err = srv.updateMechanism(nsmConnection, request, dp, extra_parameters)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Preparing to program dataplane: %v...", dp)
	dataplaneClient, dataplaneConn, err := srv.serviceRegistry.DataplaneConnection(dp)
	if err != nil {
		return nil, err
	}
	if dataplaneConn != nil { // Required for testing
		defer func() {
			err := dataplaneConn.Close()
			if err != nil {
				logrus.Errorf("Error during close Dataplane connection: %v", err)
			}
		}()
	}

	ignore_endpoints := map[string]*registry.NSERegistration{}

	var clientConnection *model.ClientConnection
	var last_error error
	var endpoint *registry.NSERegistration
	for {
		// Clone Connection to support iteration via EndPoints
		nseConnection := srv.cloneConnection(request, nsmConnection)

		endpoint, err = srv.getEndpoint(ctx, nseConnection, ignore_endpoints)
		if err != nil {
			if last_error != nil {
				return nil, fmt.Errorf("%v. Last NSE Error: %v", err, last_error)
			}
			return nil, err
		}
		// Update Request with exclude_prefixes
		srv.updateExcludePrefixes(nseConnection)
		clientConnection, err = srv.performNSERequest(request, endpoint, ctx, nseConnection, dp)

		if err != nil {
			logrus.Errorf("NSE respond with error: %v ", err)
			last_error = err
			ignore_endpoints[endpoint.NetworkserviceEndpoint.EndpointName] = endpoint
			continue
		}
		break
	}

	logrus.Infof("Sending request to dataplane: %v", clientConnection.Xcon)
	clientConnection.Xcon, err = dataplaneClient.Request(ctx, clientConnection.Xcon)
	if err != nil {
		logrus.Errorf("Dataplane request failed: %s", err)
		return nil, err
	}
	srv.model.AddClientConnection(clientConnection)
	if request.IsRemote() {
		nsmConnection = clientConnection.Xcon.GetSource().(*crossconnect.CrossConnect_RemoteSource).RemoteSource
	} else {
		nsmConnection = clientConnection.Xcon.GetSource().(*crossconnect.CrossConnect_LocalSource).LocalSource
	}
	logrus.Info("Dataplane configuration done...")
	return nsmConnection, nil
}

func (srv *networkServiceManager) cloneConnection(request nsm.NSMRequest, response nsm.NSMConnection) nsm.NSMConnection {
	var requestConnection nsm.NSMConnection
	if request.IsRemote() {
		if response == nil {
			requestConnection = proto.Clone(request.(*remote_networkservice.NetworkServiceRequest).Connection).(*remote_connection.Connection)
		} else {
			requestConnection = proto.Clone(response.(*remote_connection.Connection)).(*remote_connection.Connection)
		}
	} else {
		if response == nil {
			requestConnection = proto.Clone(request.(*networkservice.NetworkServiceRequest).Connection).(*connection.Connection)
		} else {
			requestConnection = proto.Clone(response.(*connection.Connection)).(*connection.Connection)
		}
	}
	return requestConnection
}

func (srv *networkServiceManager) Close(ctx context.Context, connection nsm.NSMConnection) error {
	logrus.Infof("Closing connection %s", connection.GetId())
	clientConnection := srv.model.DeleteClientConnection(connection.GetId())
	if clientConnection == nil {
		return fmt.Errorf("No connection with id: %s, nothing to close", connection.GetId())
	}
	var nseClientError error
	var nseCloseError error

	client, nseClientError := srv.createNSEClient(clientConnection.Endpoint)

	if client != nil {
		defer func() {
			err := client.Cleanup()
			if err != nil {
				logrus.Errorf("Error during Cleanup: %v", err)
			}
		}()
		ld := clientConnection.Xcon.GetLocalDestination()
		if ld != nil {
			nseCloseError = client.Close(ctx, ld)
		}
		rd := clientConnection.Xcon.GetRemoteDestination()
		if rd != nil {
			nseCloseError = client.Close(ctx, rd)
		}
	} else {
		logrus.Errorf("Failed to create NSE Client %v", nseClientError)
	}

	dpCloseError := srv.closeDataplane(clientConnection)
	if nseClientError != nil || nseCloseError != nil || dpCloseError != nil {
		return fmt.Errorf("Close error: %v", []error{ nseClientError, nseCloseError, dpCloseError})
	}
	return nil
}

func (srv *networkServiceManager) CreateNSERequest(endpoint *registry.NSERegistration, requestConnection nsm.NSMConnection, dataplane *model.Dataplane) nsm.NSMRequest {
	if srv.isLocalEndpoint(endpoint) {
		message := srv.createLocalNSERequest(endpoint, requestConnection)
		return message
	} else {
		message := srv.createRemoteNSMRequest(endpoint, requestConnection, dataplane)
		return message
	}
}

func (srv *networkServiceManager) isLocalEndpoint(endpoint *registry.NSERegistration) bool {
	return srv.getNetworkServiceManagerName() == endpoint.GetNetworkServiceManager().GetName()
}

func (srv *networkServiceManager) createNSEClient(endpoint *registry.NSERegistration) (nsm.NetworkServiceClient, error) {
	if srv.isLocalEndpoint(endpoint) {
		client, conn, err := srv.serviceRegistry.EndpointConnection(endpoint)
		if err != nil {
			return nil, err
		}
		return &endpointClient{connection: conn, client: client}, nil
	} else {
		client, conn, err := srv.serviceRegistry.RemoteNetworkServiceClient(endpoint.GetNetworkServiceManager())
		if err != nil {
			return nil, err
		}
		return &nsmClient{client: client, connection: conn}, nil
	}
}

func (srv *networkServiceManager) performNSERequest(request nsm.NSMRequest, endpoint *registry.NSERegistration, ctx context.Context, requestConnection nsm.NSMConnection, dp *model.Dataplane) (*model.ClientConnection, error) {
	client, err := srv.createNSEClient(endpoint)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := client.Cleanup()
		if err != nil {
			logrus.Errorf("Error during Cleanup: %v", err)
		}
	}()

	message := srv.CreateNSERequest(endpoint, requestConnection, dp)
	nseConnection, e := client.Request(ctx, message)

	if e != nil {
		logrus.Errorf("error requesting networkservice from %+v with message %#v error: %s", endpoint, message, e)
		return nil, e
	}

	err = srv.validateNSEConnection(nseConnection)
	if err != nil {
		return nil, err
	}

	err = requestConnection.UpdateContext(nseConnection.GetContext())
	if err != nil {
		err = fmt.Errorf("failure Validating NSE Connection: %s", err)
		return nil, err
	}
	srv.updateConnectionParameters(nseConnection, endpoint)

	dpApiConnection := srv.createCrossConnect(requestConnection, endpoint, request, nseConnection)
	clientConnection := &model.ClientConnection{
		ConnectionId: requestConnection.GetId(),
		Xcon:         dpApiConnection,
		Endpoint:     endpoint,
		Dataplane:    dp,
	}
	return clientConnection, nil
}

func (srv *networkServiceManager) createCrossConnect(requestConnection nsm.NSMConnection, endpoint *registry.NSERegistration, request nsm.NSMRequest, nseConnection nsm.NSMConnection) *crossconnect.CrossConnect {
	dpApiConnection := &crossconnect.CrossConnect{
		Id:      requestConnection.GetId(),
		Payload: endpoint.GetNetworkService().GetPayload(),
	}

	// We handling request from remote NSM
	if request.IsRemote() {
		dpApiConnection.Source = &crossconnect.CrossConnect_RemoteSource{
			RemoteSource: requestConnection.(*remote_connection.Connection),
		}
	} else {
		dpApiConnection.Source = &crossconnect.CrossConnect_LocalSource{
			LocalSource: requestConnection.(*connection.Connection),
		}
	}

	// We handling request from local or remote endpoint.
	//TODO: in case of remote NSE( different cluster case, this method should be changed)
	if !srv.isLocalEndpoint(endpoint) {
		dpApiConnection.Destination = &crossconnect.CrossConnect_RemoteDestination{
			RemoteDestination: nseConnection.(*remote_connection.Connection),
		}
	} else {
		dpApiConnection.Destination = &crossconnect.CrossConnect_LocalDestination{
			LocalDestination: nseConnection.(*connection.Connection),
		}
	}
	return dpApiConnection
}
func (srv *networkServiceManager) validateNSEConnection(nseConnection nsm.NSMConnection) error {
	err := nseConnection.IsComplete()
	if err != nil {
		err = fmt.Errorf("failure Validating NSE Connection: %s", err)
		return err
	}
	return nil
}

func (srv *networkServiceManager) createConnectionId() string {
	return srv.model.ConnectionId()
}

func (srv *networkServiceManager) closeDataplane(clientConnection *model.ClientConnection) error {
	logrus.Info("Closing cross connection on dataplane...")
	dataplaneClient, conn, err := srv.serviceRegistry.DataplaneConnection(clientConnection.Dataplane)
	if err != nil {
		logrus.Error(err)
		return err
	}
	if conn != nil {
		defer conn.Close()
	}
	if _, err := dataplaneClient.Close(context.Background(), clientConnection.Xcon); err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Info("Cross connection successfully closed on dataplane")
	return nil
}

func (srv *networkServiceManager) getNetworkServiceManagerName() string {
	return srv.model.GetNsm().GetName()
}

func (srv *networkServiceManager) updateConnectionParameters(nseConnection nsm.NSMConnection, endpoint *registry.NSERegistration) {
	if srv.getNetworkServiceManagerName() == endpoint.GetNetworkserviceEndpoint().GetEndpointName() {
		workspace := nsmd.WorkSpaceRegistry().WorkspaceByEndpoint(endpoint.GetNetworkserviceEndpoint())
		if workspace != nil { // In case of tests this could be empty
			nseConnection.(*connection.Connection).GetMechanism().GetParameters()[connection.Workspace] = workspace.Name()
		}
	}
}

func (srv *networkServiceManager) updateExcludePrefixes(requestConnection nsm.NSMConnection) {
	c := requestConnection.GetContext()
	if c == nil {
		c = &connectioncontext.ConnectionContext{}
	}
	for _, ep := range srv.excluded_prefixes {
		c.ExcludedPrefixes = append(c.ExcludedPrefixes, ep)
	}
	// Since we do not worry about validation, just
	requestConnection.SetContext(c)
}

func (srv *networkServiceManager) getEndpoint(ctx context.Context, requestConnection nsm.NSMConnection, ignore_endpoints map[string]*registry.NSERegistration) (*registry.NSERegistration, error) {

	// Handle case we are remote NSM and asked for particular endpoint to connect to.
	targetEndpoint := requestConnection.GetNetworkServiceEndpointName()
	if len(targetEndpoint) > 0 {
		endpoint := srv.model.GetEndpoint(targetEndpoint)
		if endpoint != nil && ignore_endpoints[endpoint.NetworkserviceEndpoint.EndpointName] == nil {
			return endpoint, nil
		} else {
			return nil, fmt.Errorf("Could not find endpoint with name: %s at local registry", targetEndpoint)
		}
	}

	// Get endpoints, do it every time since we do not know if list are changed or not.
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
	endpoints := srv.filterEndpoints(endpointResponse.GetNetworkServiceEndpoints(), ignore_endpoints)

	if len(endpoints) == 0 {
		return nil, fmt.Errorf("Failed to find NSE for NetworkService %s. Checked: %d of total NSEs: %d",
			requestConnection.GetNetworkService(), len(ignore_endpoints), len(endpoints))
	}

	endpoint := srv.model.GetSelector().SelectEndpoint(requestConnection.(*connection.Connection), endpointResponse.GetNetworkService(), endpoints)
	if endpoint == nil {
		return nil, err
	}
	return &registry.NSERegistration{
		NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[endpoint.GetNetworkServiceManagerName()],
		NetworkserviceEndpoint: endpoint,
		NetworkService:         endpointResponse.GetNetworkService(),
	}, nil
}

func (srv *networkServiceManager) filterEndpoints(endpoints []*registry.NetworkServiceEndpoint, ignore_endpoints map[string]*registry.NSERegistration) []*registry.NetworkServiceEndpoint {
	result := []*registry.NetworkServiceEndpoint{}
	// Do filter of endpoints
	for _, candidate := range endpoints {
		if ignore_endpoints[candidate.GetEndpointName()] == nil {
			result = append(result, candidate)
		}
	}
	return result
}

func (srv *networkServiceManager) filterRegEndpoints(endpoints []*registry.NSERegistration, ignore_endpoints map[string]*registry.NSERegistration) []*registry.NSERegistration {
	result := []*registry.NSERegistration{}
	// Do filter of endpoints
	for _, candidate := range endpoints {
		if ignore_endpoints[candidate.GetNetworkserviceEndpoint().GetEndpointName()] == nil {
			result = append(result, candidate)
		}
	}
	return result
}
