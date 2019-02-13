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
	"time"
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

func (srv *networkServiceManager) Request(ctx context.Context, request nsm.NSMRequest) (nsm.NSMConnection, error) {
	// Check if we are recovering connection, by checking passed connection Id is known to us.
	return srv.request(ctx, request, srv.model.GetClientConnection(request.GetConnectionId()))
}

func (srv *networkServiceManager) request(ctx context.Context, request nsm.NSMRequest, existingConnection *model.ClientConnection) (nsm.NSMConnection, error) {
	// 0. Make sure its a valid request
	err := request.IsValid()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	// 1. Create a new connection object.
	nsmConnection := srv.newConnection(request)

	// 2. Set connection id for new connections.
	// Every NSMD manage it's connections.
	if existingConnection == nil {
		nsmConnection.SetId(srv.createConnectionId())
	} else {
		// 2.1 we have connection updata/heal no need for new connection id
		nsmConnection.SetId(existingConnection.GetId())
	}

	// 3. get dataplane
	dp, err := srv.model.SelectDataplane()
	if err != nil {
		return nil, err
	}

	// A flag if we heal to close Dataplane in case of no NSE is found or failed to establish new connection.
	closeDataplaneOnNSEFailed := false

	// 4. Check if Heal/Update if we need to ask remote NSM or it is a just local mechanism change requested.
	// true if we detect we need to request NSE to upgrade/update connection.
	requestNSEOnUpdate := false
	if existingConnection != nil {
		// 4.1 New Network service is requested, we need to close current connection and do re-request of NSE.
		if nsmConnection.GetNetworkService() != existingConnection.GetNetworkService() {
			requestNSEOnUpdate = true
			closeDataplaneOnNSEFailed = true
			// Network service is closing, we need to close remote NSM and re-programm local one.
			if err := srv.close(ctx, existingConnection, false); err != nil {
				logrus.Errorf("Error during close of NSE during Request.Upgrade %v Existing connection: %v error %v", request, existingConnection, err)
			}
		} else {
			// 4.2 Check if NSE is still required, if some more context requests are different.
			requestNSEOnUpdate = srv.checkNeedNSERequest(nsmConnection, existingConnection, dp)
		}
	}

	// 5. Select a local dataplane and put it into nsmConnection object
	err = srv.updateMechanism(nsmConnection, request, dp)
	if err != nil {
		// 5.1 Close Datplane connection, if had existing one and NSE is closed.
		if closeDataplaneOnNSEFailed {
			srv.closeDataplaneLog(existingConnection)
		}
		return nil, err
	}

	// 6. Prepare dataplane connection is fine.
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

	var clientConnection *model.ClientConnection = existingConnection

	// 7. do a Request() on NSE and select it.
	if existingConnection == nil || requestNSEOnUpdate {
		//7.1 try find NSE and do a Request to it.
		clientConnection, err = srv.findConnectNSE(ctx, ignore_endpoints, request, nsmConnection, existingConnection, dp)
		if err != nil {
			if closeDataplaneOnNSEFailed {
				// 7.1.x We are failed to find NSE, and we need to close local dataplane in case of recovery.
				srv.closeDataplaneLog(existingConnection)
			}
			return nil, err
		}
	} else if existingConnection != nil {
		// 7.2 We do not need to access NSE, since all parameters are same.
		if request.IsRemote() {
			// 7.2.1 We are called from remote NSM so just copy.
			rs := clientConnection.Xcon.GetRemoteSource()
			rs.Mechanism = nsmConnection.(*remote_connection.Connection).Mechanism
			rs.State = remote_connection.State_UP
		} else {
			// 7.2.2 It is local connection from NSC, so just copy values.
			ls := clientConnection.Xcon.GetLocalSource()
			ls.Mechanism = nsmConnection.(*connection.Connection).Mechanism
			ls.State = connection.State_UP
		}
	}

	// 8. Remember original Request for Heal cases.
	if existingConnection == nil {
		clientConnection.Request = request
	}

	// 9. We need Add connection to model, or update it in case of Healing.
	if existingConnection == nil {
		srv.model.AddClientConnection(clientConnection)
	}

	// 10. We need to programm dataplane with our values.
	// 10.1 TODO: Close current dataplane local configuration, since currently Dataplane doesn't support upgrade.
	if existingConnection != nil {
		if _, err := dataplaneClient.Close(ctx, existingConnection.Xcon); err != nil {
			logrus.Errorf("Closing Dataplane error for local connection: %v", err)
		}
	}
	// 10.2 Sending updated request to dataplane.
	logrus.Infof("Sending request to dataplane: %v", clientConnection.Xcon)
	clientConnection.Xcon, err = dataplaneClient.Request(ctx, clientConnection.Xcon)
	if err != nil {
		logrus.Errorf("Dataplane request failed: %s", err)
		// Let's try again with a short delay
		<-time.Tick(500)
		logrus.Errorf("Dataplane request retry: %v", clientConnection.Xcon)
		clientConnection.Xcon, err = dataplaneClient.Request(ctx, clientConnection.Xcon)

		if err != nil {
			logrus.Errorf("Dataplane request retry failed: %s", err)
			// 10.3 If datplane configuration are failed, we need to close remore NSE actually.
			if dp_err := srv.close(context.Background(), clientConnection, true); dp_err != nil {
				logrus.Errorf("Failed to NSE.Close() caused by local dataplane configuration failure.")
			}
			// 10.4 We need to remove local connection we just added already.
			srv.model.DeleteClientConnection(clientConnection.ConnectionId)
			return nil, err
		}
	}
	logrus.Infof("Dataplane configuration sucessfull %v", clientConnection.Xcon)

	// 11. Send update for client connection
	clientConnection.ConnectionState = model.ClientConnection_Ready
	if existingConnection != nil {
		srv.model.UpdateClientConnection(clientConnection)
	}

	// 11. We are done with configuration here.
	if request.IsRemote() {
		nsmConnection = clientConnection.Xcon.GetSource().(*crossconnect.CrossConnect_RemoteSource).RemoteSource
	} else {
		nsmConnection = clientConnection.Xcon.GetSource().(*crossconnect.CrossConnect_LocalSource).LocalSource
	}
	logrus.Info("Dataplane configuration done...")
	return nsmConnection, nil
}

func (srv *networkServiceManager) waitRemoteUpdateEvent(existingConnection *model.ClientConnection) {
	if existingConnection.RemoteNsm == nil {
		// No need to wait, since there is no remote part
		return
	}
	st := time.Now()
	for !existingConnection.UpdateRecieved && time.Since(st) < 10*time.Second {
		// Wait for update event to arrive
		logrus.Infof("Waiting update event to arrive... %v", existingConnection)
		<-time.Tick(10000 * time.Millisecond)
	}
}

func (srv *networkServiceManager) closeDataplaneLog(existingConnection *model.ClientConnection) {
	if dp_err := srv.closeDataplane(existingConnection); dp_err != nil {
		logrus.Errorf("Failed to close local Dataplane for connection %v", existingConnection)
	}
}

func (srv *networkServiceManager) findConnectNSE(ctx context.Context, ignore_endpoints map[string]*registry.NSERegistration, request nsm.NSMRequest, nsmConnection nsm.NSMConnection, existingConnection *model.ClientConnection, dp *model.Dataplane) (*model.ClientConnection, error) {
	var endpoint *registry.NSERegistration
	var err error
	var last_error error
	var clientConnection *model.ClientConnection
	for {
		endpoint = nil
		// 7.1.1 Clone Connection to support iteration via EndPoints
		nseConnection := srv.cloneConnection(request, nsmConnection)

		if existingConnection != nil {
			// 7.2.1 Check previous endpoint, and it we will be able to contact it, it should be fine.
			if ignore_endpoints[existingConnection.Endpoint.NetworkserviceEndpoint.EndpointName] == nil {
				endpoint = existingConnection.Endpoint
			}
		}
		// 7.2.2 Check if endpoint is not ignored yet

		if endpoint == nil {
			// 7.2.3 Choose a new endpoint
			endpoint, err = srv.getEndpoint(ctx, nseConnection, ignore_endpoints)
		}
		if err != nil {
			// 7.2.4 No endpoints found, we need to return error, including last error for previous NSE
			if last_error != nil {
				return nil, fmt.Errorf("%v. Last NSE Error: %v", err, last_error)
			} else {
				return nil, err
			}
		}
		// 7.2.5 Update Request with exclude_prefixes, etc
		srv.updateExcludePrefixes(nseConnection)

		// 7.2.6 perform request to NSE/remote NSMD/NSE
		clientConnection, err = srv.performNSERequest(ctx, endpoint, nseConnection, request, dp, existingConnection)

		// 7.2.7 in case of error we put NSE into ignored list to check another one.
		if err != nil {
			logrus.Errorf("NSE respond with error: %v ", err)
			last_error = err
			ignore_endpoints[endpoint.NetworkserviceEndpoint.EndpointName] = endpoint
			continue
		}
		// 7.2.8 If we requesting existing NSE on Remote NSM, we need to wait for Update event
		if existingConnection != nil && endpoint == existingConnection.Endpoint {
			// We need to wait for update event to be recieved
			if !srv.isLocalEndpoint(clientConnection.Endpoint) {
				srv.waitRemoteUpdateEvent(existingConnection)
			}
		}

		// 7.2.9 We are fine with NSE connection and could continue.
		return clientConnection, nil
	}
}

func (srv *networkServiceManager) cloneConnection(request nsm.NSMRequest, response nsm.NSMConnection) nsm.NSMConnection {
	var requestConnection nsm.NSMConnection
	if request.IsRemote() {
		requestConnection = proto.Clone(response.(*remote_connection.Connection)).(*remote_connection.Connection)
	} else {
		requestConnection = proto.Clone(response.(*connection.Connection)).(*connection.Connection)
	}
	return requestConnection
}
func (srv *networkServiceManager) newConnection(request nsm.NSMRequest) nsm.NSMConnection {
	var requestConnection nsm.NSMConnection
	if request.IsRemote() {
		requestConnection = proto.Clone(request.(*remote_networkservice.NetworkServiceRequest).Connection).(*remote_connection.Connection)
	} else {
		requestConnection = proto.Clone(request.(*networkservice.NetworkServiceRequest).Connection).(*connection.Connection)
	}
	return requestConnection
}

func (srv *networkServiceManager) Close(ctx context.Context, connection nsm.NSMClientConnection) error {
	return srv.close(ctx, connection.(*model.ClientConnection), true)
}

func (srv *networkServiceManager) close(ctx context.Context, clientConnection *model.ClientConnection, closeDataplane bool) error {
	logrus.Infof("Closing connection %v", clientConnection)
	if clientConnection.ConnectionState == model.ClientConnection_Closing {
		return nil
	}
	clientConnection.ConnectionState = model.ClientConnection_Closing
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
	var dpCloseError error = nil
	if closeDataplane {
		dpCloseError = srv.closeDataplane(clientConnection)
		// TODO: We need to be sure Dataplane is respond well so we could delete connection.
		srv.model.DeleteClientConnection(clientConnection.ConnectionId)
	}
	clientConnection.ConnectionState = model.ClientConnection_Closed

	if nseClientError != nil || nseCloseError != nil || dpCloseError != nil {
		return fmt.Errorf("Close error: %v", []error{nseClientError, nseCloseError, dpCloseError})
	}
	return nil
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

func (srv *networkServiceManager) performNSERequest(ctx context.Context, endpoint *registry.NSERegistration, requestConnection nsm.NSMConnection, request nsm.NSMRequest, dp *model.Dataplane, existingConnection *model.ClientConnection) (*model.ClientConnection, error) {
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

	var message nsm.NSMRequest
	if srv.isLocalEndpoint(endpoint) {
		message = srv.createLocalNSERequest(endpoint, requestConnection)
	} else {
		message = srv.createRemoteNSMRequest(endpoint, requestConnection, dp, existingConnection)
	}
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
	if !srv.isLocalEndpoint(endpoint) {
		clientConnection.RemoteNsm = endpoint.GetNetworkServiceManager()
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
	if srv.isLocalEndpoint(endpoint) {
		workspace := nsmd.WorkSpaceRegistry().WorkspaceByEndpoint(endpoint.GetNetworkserviceEndpoint())
		if workspace != nil { // In case of tests this could be empty
			nseConnection.(*connection.Connection).GetMechanism().GetParameters()[connection.Workspace] = workspace.Name()
		}
		logrus.Infof("Update Local NSE connection parameters: %v", nseConnection.(*connection.Connection).GetMechanism())
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

/**
check if we need to do a NSE/Remote NSM request in case of our connection Upgrade/Healing procedure.
*/
func (srv *networkServiceManager) checkNeedNSERequest(nsmConnection nsm.NSMConnection, existingConnection *model.ClientConnection, dp *model.Dataplane) bool {
	// Check if context is changed, if changed we need to
	if !proto.Equal(nsmConnection.GetContext(), existingConnection.GetSourceConnection().GetContext()) {
		return true
	}
	// We need to check, dp has mechanism changes in our Remote connection selected mechanism.

	if remoteDestination := existingConnection.Xcon.GetRemoteDestination(); remoteDestination != nil {
		// Let's check if remote destination is matchs our dataplane destination.
		if dpM := findRemoteMechanism(dp.RemoteMechanisms, remoteDestination.GetMechanism().GetType()); dpM != nil {
			// We need to check if source mechanism type and source parameters are different
			for k, v := range dpM.Parameters {
				rmV := remoteDestination.Mechanism.Parameters[k]
				if v != rmV {
					logrus.Infof("Remote mechanism parameter %s was different with previous one : %v  %v", k, rmV, v)
					return true
				}
			}
			if !proto.Equal(dpM, remoteDestination.Mechanism) {
				logrus.Infof("Remote mechanism was different with previous selected one : %v  %v", remoteDestination.Mechanism, dpM)
				return true
			}
		} else {
			logrus.Infof("Remote mechanism previously selected was not found: %v  in dataplane %v", remoteDestination.Mechanism, dp.RemoteMechanisms)
			return true
		}
	}

	return false
}
