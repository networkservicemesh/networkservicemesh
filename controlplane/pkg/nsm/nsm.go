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
	"crypto/rand"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"sync"
	"time"
)

const (
	DataplaneRetryCount = 10 // A number of times to call Dataplane Request, TODO: Remove after DP will be stable.
	DataplaneRetryDelay = 500 * time.Millisecond
	DataplaneTimeout    = 15 * time.Second
)

// Network service manager to manage both local/remote NSE connections.
type networkServiceManager struct {
	sync.RWMutex
	serviceRegistry  serviceregistry.ServiceRegistry
	model            model.Model
	excludedPrefixes prefix_pool.PrefixPool
	properties       *nsm.NsmProperties
	stateRestored    chan bool
	errCh            chan error

	healProcessor networkServiceHealProcessor
	nseManager    networkServiceEndpointManager
}

func (srv *networkServiceManager) GetHealProperties() *nsm.NsmProperties {
	return srv.properties
}

func NewNetworkServiceManager(model model.Model, serviceRegistry serviceregistry.ServiceRegistry) nsm.NetworkServiceManager {
	emptyPrefixPool, _ := prefix_pool.NewPrefixPool()
	properties := nsm.NewNsmProperties()
	nseManager := &nseManager{
		serviceRegistry: serviceRegistry,
		model:           model,
		properties:      properties,
	}

	srv := &networkServiceManager{
		serviceRegistry:  serviceRegistry,
		model:            model,
		excludedPrefixes: emptyPrefixPool,
		properties:       properties,
		stateRestored:    make(chan bool, 1),
		errCh:            make(chan error, 1),

		nseManager: nseManager,
	}

	srv.healProcessor = &nsmHealProcessor{
		serviceRegistry: serviceRegistry,
		model:           model,
		properties:      properties,

		conManager: srv,
		nseManager: nseManager,
	}

	go srv.monitorExcludePrefixes()
	return srv
}

func (srv *networkServiceManager) monitorExcludePrefixes() {
	poolCh, err := GetExcludedPrefixes(srv.serviceRegistry)
	if err != nil {
		srv.errCh <- err
		return
	}

	for {
		pool, ok := <-poolCh
		if !ok {
			srv.errCh <- fmt.Errorf("nsmd-k8s is not responding, exclude prefixes won't be updating")
			return
		}

		srv.Lock()
		srv.excludedPrefixes = pool
		srv.Unlock()
	}
}

func (srv *networkServiceManager) Request(ctx context.Context, request nsm.NSMRequest) (nsm.NSMConnection, error) {
	// Check if we are recovering connection, by checking passed connection Id is known to us.
	return srv.request(ctx, request, srv.model.GetClientConnection(request.GetConnectionId()))
}

func create_logid() (uuid string) {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		return
	}

	uuid = fmt.Sprintf("%X", b[0:4])
	return
}

func (srv *networkServiceManager) request(ctx context.Context, request nsm.NSMRequest, existingConnection *model.ClientConnection) (nsm.NSMConnection, error) {
	requestId := create_logid()
	logrus.Infof("NSM:(%v) request: %v", requestId, request)
	if existingConnection != nil {
		logrus.Infof("NSM:(%v) Called with existing connection passed: %v", requestId, existingConnection)
	}

	// 0. Make sure its a valid request
	err := request.IsValid()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	// 1. Create a new connection object.
	nsmConnection := newConnection(request)

	// 2. Set connection id for new connections.
	// Every NSMD manage it's connections.
	if existingConnection == nil {
		nsmConnection.SetId(srv.createConnectionId())
	} else {
		// 2.1 we have connection updata/heal no need for new connection id
		nsmConnection.SetId(existingConnection.GetId())
	}

	// 3. get dataplane
	srv.serviceRegistry.WaitForDataplaneAvailable(srv.model, DataplaneTimeout)
	dp, err := srv.model.SelectDataplane(func(dp *model.Dataplane) bool {
		if request.IsRemote() {
			return findRemoteMechanism(dp.RemoteMechanisms, remote_connection.MechanismType_VXLAN) != nil
		} else {
			r := request.(*local_networkservice.NetworkServiceRequest)
			for _, m := range r.MechanismPreferences {
				if findLocalMechanism(dp.LocalMechanisms, m.Type) != nil {
					return true
				}
			}
		}
		return false
	})

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
			if err := srv.close(ctx, existingConnection, false, false); err != nil {
				logrus.Errorf("NSM:(4.1-%v) Error during close of NSE during Request.Upgrade %v Existing connection: %v error %v", requestId, request, existingConnection, err)
			}
		} else {
			// 4.2 Check if NSE is still required, if some more context requests are different.
			requestNSEOnUpdate = srv.checkNeedNSERequest(requestId, nsmConnection, existingConnection, dp)
		}
	}

	// 5. Select a local dataplane and put it into nsmConnection object
	err = srv.updateMechanism(requestId, nsmConnection, request, dp)
	if err != nil {
		// 5.1 Close Datplane connection, if had existing one and NSE is closed.
		if closeDataplaneOnNSEFailed {
			if dp_err := srv.closeDataplane(existingConnection); dp_err != nil {
				logrus.Errorf("NSM:(5.1-%v) Failed to close local Dataplane for connection %v", requestId, existingConnection)
			}
		}
		return nil, err
	}

	// 6. Prepare dataplane connection is fine.
	logrus.Infof("NSM:(6-%v) Preparing to program dataplane: %v...", requestId, dp)
	dataplaneClient, dataplaneConn, err := srv.serviceRegistry.DataplaneConnection(dp)
	if err != nil {
		return nil, err
	}
	if dataplaneConn != nil { // Required for testing
		defer func() {
			err := dataplaneConn.Close()
			if err != nil {
				logrus.Errorf("NSM:(6.1-%v) Error during close Dataplane connection: %v", requestId, err)
			}
		}()
	}

	ignore_endpoints := map[string]*registry.NSERegistration{}

	var clientConnection *model.ClientConnection = existingConnection

	// 7. do a Request() on NSE and select it.
	if existingConnection == nil || requestNSEOnUpdate {
		//7.1 try find NSE and do a Request to it.
		clientConnection, err = srv.findConnectNSE(requestId, ctx, ignore_endpoints, request, nsmConnection, existingConnection, dp)
		if err != nil {
			if closeDataplaneOnNSEFailed {
				// 7.1.x We are failed to find NSE, and we need to close local dataplane in case of recovery.
				if dp_err := srv.closeDataplane(existingConnection); dp_err != nil {
					logrus.Errorf("NSM:(7.1-%v) Failed to close local Dataplane for connection %v", requestId, existingConnection)
				}
			}
			if existingConnection != nil {
				srv.model.DeleteClientConnection(existingConnection.ConnectionId)
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
			ls.Mechanism = nsmConnection.(*local_connection.Connection).Mechanism
			ls.State = local_connection.State_UP
		}
	}

	// 8. Remember original Request for Heal cases.
	clientConnection.Request = request

	// 9. We need Add connection to model, or update it in case of Healing.
	if existingConnection == nil {
		srv.model.AddClientConnection(clientConnection)
	}

	// 10. We need to programm dataplane with our values.
	// 10.1 TODO: Close current dataplane local configuration, since currently Dataplane doesn't support upgrade.
	if existingConnection != nil {
		logrus.Errorf("NSM:(10.0-%v) Closing Dataplane because of existing connection passed...", requestId)
		if err := srv.closeDataplane(existingConnection); err != nil {
			logrus.Errorf("NSM:(10.1-%v) Closing Dataplane error for local connection: %v", requestId, err)
		}
	}
	// 10.2 Sending updated request to dataplane.
	for dpRetry := 0; dpRetry < DataplaneRetryCount; dpRetry++ {
		if err := ctx.Err(); err != nil {
			srv.handleDataplaneContextTimeout(requestId, err, clientConnection)
			return nil, ctx.Err()
		}

		logrus.Infof("NSM:(10.2-%v) Sending request to dataplane: %v retry: %v", requestId, clientConnection.Xcon, dpRetry)
		dpCtx, cancel := context.WithTimeout(context.Background(), DataplaneTimeout)
		defer cancel()
		newXcon, err := dataplaneClient.Request(dpCtx, clientConnection.Xcon)
		if err != nil {
			logrus.Errorf("NSM:(10.2.1-%v) Dataplane request failed: %v retry: %v", requestId, err, dpRetry)

			// Let's try again with a short delay
			if dpRetry < DataplaneRetryCount-1 {
				<-time.Tick(DataplaneRetryDelay)

				if dp_err := srv.closeDataplane(clientConnection); dp_err != nil {
					logrus.Errorf("NSM:(10.2.4-%v) Failed to NSE.Close() caused by local dataplane configuration failure: %v", requestId, dp_err)
				}
				continue
			}
			logrus.Errorf("NSM:(10.2.2-%v) Dataplane request  all retry attempts failed: %v", requestId, clientConnection.Xcon)
			// 10.3 If datplane configuration are failed, we need to close remore NSE actually.
			if dp_err := srv.close(context.Background(), clientConnection, false, false); dp_err != nil {
				logrus.Errorf("NSM:(10.2.4-%v) Failed to NSE.Close() caused by local dataplane configuration failure: %v", requestId, dp_err)
			}
			// 10.4 We need to remove local connection we just added already.
			srv.model.DeleteClientConnection(clientConnection.ConnectionId)
			return nil, err
		}
		clientConnection.Xcon = newXcon

		// In case of context deadline, we need to close NSE and dataplane.
		if err := ctx.Err(); err != nil {
			srv.handleDataplaneContextTimeout(requestId, err, clientConnection)
			return nil, ctx.Err()
		}

		logrus.Infof("NSM:(10.3-%v) Dataplane configuration successful %v", requestId, clientConnection.Xcon)
		break
	}

	// 11. Send update for client connection
	clientConnection.ConnectionState = model.ClientConnection_Ready
	clientConnection.DataplaneState = model.DataplaneState_Ready
	if existingConnection != nil {
		srv.model.UpdateClientConnection(clientConnection)
	}

	// 11. We are done with configuration here.
	if request.IsRemote() {
		nsmConnection = clientConnection.Xcon.GetSource().(*crossconnect.CrossConnect_RemoteSource).RemoteSource
	} else {
		nsmConnection = clientConnection.Xcon.GetSource().(*crossconnect.CrossConnect_LocalSource).LocalSource
	}
	logrus.Infof("NSM:(11-%v) Request done...", requestId)
	return nsmConnection, nil
}

func (srv *networkServiceManager) handleDataplaneContextTimeout(requestId string, err error, clientConnection *model.ClientConnection) {
	logrus.Errorf("NSM:(10.2.0-%v) Context timeout, during programming Dataplane... %v", requestId, err)
	// If context is exceed
	if ep_err := srv.closeEndpoint(context.Background(), clientConnection); ep_err != nil {
		logrus.Errorf("NSM:(10.2.0-%v) Context timeout, closing NSE: %v", requestId, ep_err)
	}
	srv.model.DeleteClientConnection(clientConnection.ConnectionId)
}

func (srv *networkServiceManager) findConnectNSE(requestId string, ctx context.Context, ignore_endpoints map[string]*registry.NSERegistration, request nsm.NSMRequest, nsmConnection nsm.NSMConnection, existingConnection *model.ClientConnection, dp *model.Dataplane) (*model.ClientConnection, error) {
	// 7.x
	var endpoint *registry.NSERegistration
	var err error
	var last_error error
	var clientConnection *model.ClientConnection
	for {
		if err := ctx.Err(); err != nil {
			logrus.Errorf("NSM:(7.1.0-%v) Context timeout, during find/call NSE... %v", requestId, err)
			return nil, err
		}
		endpoint = nil
		// 7.1.1 Clone Connection to support iteration via EndPoints
		nseConnection := cloneConnection(request, nsmConnection)

		if existingConnection != nil {
			// 7.1.2 Check previous endpoint, and it we will be able to contact it, it should be fine.
			if existingConnection.Endpoint != nil && ignore_endpoints[existingConnection.Endpoint.NetworkserviceEndpoint.EndpointName] == nil {
				endpoint = existingConnection.Endpoint
			}
		}
		// 7.1.3 Check if endpoint is not ignored yet

		if endpoint == nil {
			// 7.1.4 Choose a new endpoint
			endpoint, err = srv.nseManager.getEndpoint(ctx, nseConnection, ignore_endpoints)
		}
		if err != nil {
			// 7.1.5 No endpoints found, we need to return error, including last error for previous NSE
			if last_error != nil {
				return nil, fmt.Errorf("NSM:(7.1.5-%v) %v. Last NSE Error: %v", requestId, err, last_error)
			} else {
				return nil, err
			}
		}
		// 7.1.6 Update Request with exclude_prefixes, etc
		srv.updateExcludePrefixes(nseConnection)

		// 7.1.7 perform request to NSE/remote NSMD/NSE
		clientConnection, err = srv.performNSERequest(requestId, ctx, endpoint, nseConnection, request, dp, existingConnection)

		// 7.1.8 in case of error we put NSE into ignored list to check another one.
		if err != nil {
			logrus.Errorf("NSM:(7.1.8-%v) NSE respond with error: %v ", requestId, err)
			last_error = err
			ignore_endpoints[endpoint.NetworkserviceEndpoint.EndpointName] = endpoint
			continue
		}
		// 7.1.9 We are fine with NSE connection and could continue.
		return clientConnection, nil
	}
}

func (srv *networkServiceManager) Close(ctx context.Context, connection nsm.NSMClientConnection) error {
	return srv.close(ctx, connection.(*model.ClientConnection), true, true)
}

func (srv *networkServiceManager) close(ctx context.Context, clientConnection *model.ClientConnection, closeDataplane bool, modelRemove bool) error {
	logrus.Infof("NSM: Closing connection %v", clientConnection)
	if clientConnection.ConnectionState == model.ClientConnection_Closing {
		return nil
	}
	clientConnection.ConnectionState = model.ClientConnection_Closing
	var nseClientError error
	var nseCloseError error

	srv.closeEndpoint(ctx, clientConnection)

	var dpCloseError error = nil
	if closeDataplane {
		dpCloseError = srv.closeDataplane(clientConnection)
		// TODO: We need to be sure Dataplane is respond well so we could delete connection.
		if modelRemove {
			srv.model.DeleteClientConnection(clientConnection.ConnectionId)
		}
	}

	if nseClientError != nil || nseCloseError != nil || dpCloseError != nil {
		return fmt.Errorf("NSM: Close error: %v", []error{nseClientError, nseCloseError, dpCloseError})
	}
	logrus.Infof("NSM: Close for %s complete...", clientConnection.GetId())
	return nil
}

func (srv *networkServiceManager) performNSERequest(requestId string, ctx context.Context, endpoint *registry.NSERegistration, requestConnection nsm.NSMConnection, request nsm.NSMRequest, dp *model.Dataplane, existingConnection *model.ClientConnection) (*model.ClientConnection, error) {
	// 7.2.6.x
	connectCtx, connectCancel := context.WithTimeout(ctx, srv.properties.RequestConnectTimeout)
	defer connectCancel()

	client, err := srv.nseManager.createNSEClient(connectCtx, endpoint)
	if err != nil {
		// 7.2.6.1
		return nil, fmt.Errorf("NSM:(7.2.6.1) Failed to create NSE Client. %v", err)
	}
	defer func() {
		err := client.Cleanup()
		if err != nil {
			logrus.Errorf("NSM:(7.2.6.2-%v) Error during Cleanup: %v", requestId, err)
		}
	}()

	var message nsm.NSMRequest
	if srv.nseManager.isLocalEndpoint(endpoint) {
		message = srv.createLocalNSERequest(endpoint, requestConnection)
	} else {
		message = srv.createRemoteNSMRequest(endpoint, requestConnection, dp, existingConnection)
	}
	logrus.Infof("NSM:(7.2.6.2-%v) Requesting NSE with request %v", requestId, message)
	nseConnection, e := client.Request(ctx, message)

	if e != nil {
		logrus.Errorf("NSM:(7.2.6.2.1-%v) error requesting networkservice from %+v with message %#v error: %s", requestId, endpoint, message, e)
		return nil, e
	}

	// 7.2.6.2.2
	err = srv.validateNSEConnection(requestId, nseConnection)
	if err != nil {
		return nil, err
	}

	// 7.2.6.2.3
	err = requestConnection.UpdateContext(nseConnection.GetContext())
	if err != nil {
		err = fmt.Errorf("NSM:(7.2.6.2.3-%v) failure Validating NSE Connection: %s", requestId, err)
		return nil, err
	}
	// 7.2.6.2.4 update connection parameters, add workspace if local nse
	srv.updateConnectionParameters(requestId, nseConnection, endpoint)

	// 7.2.6.2.5 create cross connection
	dpApiConnection := srv.createCrossConnect(requestConnection, endpoint, request, nseConnection)
	clientConnection := &model.ClientConnection{
		ConnectionId:    requestConnection.GetId(),
		Xcon:            dpApiConnection,
		Endpoint:        endpoint,
		Dataplane:       dp,
		ConnectionState: model.ClientConnection_Requesting,
	}
	// 7.2.6.2.6 - It not a local NSE put remote NSM name in request
	if !srv.nseManager.isLocalEndpoint(endpoint) {
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
			LocalSource: requestConnection.(*local_connection.Connection),
		}
	}

	// We handling request from local or remote endpoint.
	//TODO: in case of remote NSE( different cluster case, this method should be changed)
	if !srv.nseManager.isLocalEndpoint(endpoint) {
		dpApiConnection.Destination = &crossconnect.CrossConnect_RemoteDestination{
			RemoteDestination: nseConnection.(*remote_connection.Connection),
		}
	} else {
		dpApiConnection.Destination = &crossconnect.CrossConnect_LocalDestination{
			LocalDestination: nseConnection.(*local_connection.Connection),
		}
	}
	return dpApiConnection
}
func (srv *networkServiceManager) validateNSEConnection(requestId string, nseConnection nsm.NSMConnection) error {
	srv.RLock()
	defer srv.RUnlock()

	errorFormatter := func(err error) error {
		return fmt.Errorf("NSM:(7.2.6.2.2-%v) failure Validating NSE Connection: %s", requestId, err)
	}

	err := nseConnection.IsComplete()
	if err != nil {
		return errorFormatter(err)
	}

	if srcIp := nseConnection.GetContext().GetSrcIpAddr(); srcIp != "" {
		intersect, err := srv.excludedPrefixes.Intersect(srcIp)
		if err != nil {
			return errorFormatter(err)
		}
		if intersect {
			return errorFormatter(fmt.Errorf("srcIp intersects excludedPrefix"))
		}
	}

	if dstIp := nseConnection.GetContext().GetDstIpAddr(); dstIp != "" {
		intersect, err := srv.excludedPrefixes.Intersect(dstIp)
		if err != nil {
			return errorFormatter(err)
		}
		if intersect {
			return errorFormatter(fmt.Errorf("dstIp intersects excludedPrefix"))
		}
	}

	return nil
}

func (srv *networkServiceManager) createConnectionId() string {
	return srv.model.ConnectionId()
}

func (srv *networkServiceManager) closeDataplane(clientConnection *model.ClientConnection) error {
	if clientConnection.DataplaneState == model.DataplaneState_None {
		// Do not need to close
		return nil
	}
	logrus.Info("NSM.Dataplane: Closing cross connection on dataplane...")
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
	logrus.Info("NSM.Dataplane: Cross connection successfully closed on dataplane")
	clientConnection.DataplaneState = model.DataplaneState_None
	return nil
}

func (srv *networkServiceManager) getNetworkServiceManagerName() string {
	return srv.model.GetNsm().GetName()
}

func (srv *networkServiceManager) updateConnectionParameters(requestId string, nseConnection nsm.NSMConnection, endpoint *registry.NSERegistration) {
	if srv.nseManager.isLocalEndpoint(endpoint) {
		modelEp := srv.model.GetEndpoint(endpoint.GetNetworkserviceEndpoint().GetEndpointName())
		if modelEp != nil { // In case of tests this could be empty
			nseConnection.(*local_connection.Connection).GetMechanism().GetParameters()[local_connection.Workspace] = modelEp.Workspace
			nseConnection.(*local_connection.Connection).GetMechanism().GetParameters()[local_connection.WorkspaceNSEName] = modelEp.Endpoint.NetworkserviceEndpoint.EndpointName
		}
		logrus.Infof("NSM:(7.2.6.2.4-%v) Update Local NSE connection parameters: %v", requestId, nseConnection.(*local_connection.Connection).GetMechanism())
	}
}

func (srv *networkServiceManager) updateExcludePrefixes(requestConnection nsm.NSMConnection) {
	c := requestConnection.GetContext()
	if c == nil {
		c = &connectioncontext.ConnectionContext{}
	}
	c.ExcludedPrefixes = append(c.ExcludedPrefixes, srv.excludedPrefixes.GetPrefixes()...)

	// Since we do not worry about validation, just
	requestConnection.SetContext(c)
}

/**
check if we need to do a NSE/Remote NSM request in case of our connection Upgrade/Healing procedure.
*/
func (srv *networkServiceManager) checkNeedNSERequest(requestId string, nsmConnection nsm.NSMConnection, existingConnection *model.ClientConnection, dp *model.Dataplane) bool {
	// 4.2.x
	// 4.2.1 Check if context is changed, if changed we need to
	if !proto.Equal(nsmConnection.GetContext(), existingConnection.GetConnectionSource().GetContext()) {
		return true
	}
	// We need to check, dp has mechanism changes in our Remote connection selected mechanism.

	if remoteDestination := existingConnection.Xcon.GetRemoteDestination(); remoteDestination != nil {
		// 4.2.2 Let's check if remote destination is matchs our dataplane destination.
		if dpM := findRemoteMechanism(dp.RemoteMechanisms, remoteDestination.GetMechanism().GetType()); dpM != nil {
			// 4.2.3 We need to check if source mechanism type and source parameters are different
			for k, v := range dpM.Parameters {
				rmV := remoteDestination.Mechanism.Parameters[k]
				if v != rmV {
					logrus.Infof("NSM:(4.2.3-%v) Remote mechanism parameter %s was different with previous one : %v  %v", requestId, k, rmV, v)
					return true
				}
			}
			if !proto.Equal(dpM, remoteDestination.Mechanism) {
				logrus.Infof("NSM:(4.2.4-%v)  Remote mechanism was different with previous selected one : %v  %v", requestId, remoteDestination.Mechanism, dpM)
				return true
			}
		} else {
			logrus.Infof("NSM:(4.2.5-%v) Remote mechanism previously selected was not found: %v  in dataplane %v", requestId, remoteDestination.Mechanism, dp.RemoteMechanisms)
			return true
		}
	}

	return false
}

func (srv *networkServiceManager) WaitForDataplane(timeout time.Duration) error {
	// Wait for at least one dataplane to be available
	if err := srv.serviceRegistry.WaitForDataplaneAvailable(srv.model, timeout); err != nil {
		return err
	}
	logrus.Infof("Dataplane is available, waiting for initial state recieved and processed...")
	select {
	case <-srv.stateRestored:
		return nil
	case <-time.Tick(timeout):
		return fmt.Errorf("Failed to wait for NSMD stare restore... timeout %v happened", timeout)
	}
}

func (srv *networkServiceManager) RestoreConnections(xcons []*crossconnect.CrossConnect, dataplane string) {
	for _, xcon := range xcons {

		// Model should increase its id counter to max of xcons restored from dataplane
		srv.model.CorrectIdGenerator(xcon.GetId())

		existing := srv.model.GetClientConnection(xcon.GetId())
		if existing == nil {
			logrus.Infof("Restoring state of active connection %v", xcon)

			endpointName := ""
			networkServiceName := ""
			var endpoint *registry.NSERegistration
			connectionState := model.ClientConnection_Ready

			dp := srv.model.GetDataplane(dataplane)

			discovery, err := srv.serviceRegistry.DiscoveryClient()
			if err != nil {
				logrus.Errorf("Failed to find NSE to recovery: %v", err)
			}
			if src := xcon.GetRemoteSource(); src != nil {
				// Since source is remote, connection need to be healed.
				connectionState = model.ClientConnection_Healing

				networkServiceName = src.GetNetworkService()
				endpointName = src.GetNetworkServiceEndpointName()
			}
			if dst := xcon.GetLocalDestination(); dst != nil {
				// Local NSE, connection is Ready
				connectionState = model.ClientConnection_Ready

				networkServiceName = dst.GetNetworkService()
				endpointName = dst.GetMechanism().GetParameters()[local_connection.WorkspaceNSEName]
			}
			if dst := xcon.GetRemoteDestination(); dst != nil {
				// NSE is remote one, and source is local one, we are ready.
				connectionState = model.ClientConnection_Ready

				networkServiceName = xcon.GetRemoteDestination().GetNetworkService()
				endpointName = xcon.GetRemoteDestination().GetNetworkServiceEndpointName()

				// In case VxLan is used we need to correct vlanId id generator.
				m := dst.GetMechanism()
				if m.Type == remote_connection.MechanismType_VXLAN {
					srcIp, err := m.SrcIP()
					dstIp, err2 := m.DstIP()
					vni, err3 := m.VNI()
					if err != nil || err2 != nil || err3 != nil {
						logrus.Errorf("Error retriving SRC/DST IP or VNI from Remote connection %v %v", err, err2)
					} else {
						srv.serviceRegistry.VniAllocator().Restore(srcIp, dstIp, vni)
					}
				}
			}

			if endpointName != "" {
				logrus.Infof("Discovering endpoint at registry Network service: %s endpoint: %s ", networkServiceName, endpointName)

				localEndpoint := srv.model.GetEndpoint(endpointName)
				if localEndpoint != nil {
					logrus.Infof("Local endpoint selected: %v", localEndpoint)
					endpoint = localEndpoint.Endpoint
				} else {
					endpoints, err := discovery.FindNetworkService(context.Background(), &registry.FindNetworkServiceRequest{
						NetworkServiceName: networkServiceName,
					})
					if err != nil {
						logrus.Errorf("Failed to find NSE to recovery: %v", err)
					}
					for _, ep := range endpoints.NetworkServiceEndpoints {
						if xcon.GetRemoteDestination() != nil && ep.EndpointName == xcon.GetRemoteDestination().GetNetworkServiceEndpointName() {
							endpoint = &registry.NSERegistration{
								NetworkServiceManager:  endpoints.NetworkServiceManagers[ep.NetworkServiceManagerName],
								NetworkserviceEndpoint: ep,
								NetworkService:         endpoints.NetworkService,
							}
							break
						}
					}
				}
				if endpoint == nil {
					logrus.Errorf("Failed to find Endpoint %s", endpointName)
				} else {
					logrus.Infof("Endpoint found: %v", endpoint)
				}
			}

			clientConnection := &model.ClientConnection{
				ConnectionId:    xcon.GetId(),
				Xcon:            xcon,
				Endpoint:        endpoint, // We do not have endpoint here.
				Dataplane:       dp,
				ConnectionState: connectionState,
				DataplaneState:  model.DataplaneState_Ready, // It is configured already.
			}
			srv.model.AddClientConnection(clientConnection)

			// Add healing timer, for connection to be headled from source side.
			if src := xcon.GetRemoteSource(); src != nil {
				if endpoint != nil {
					srv.RemoteConnectionLost(clientConnection)
				} else {
					srv.closeLocalMissingNSE(clientConnection)
				}
			} else if src := xcon.GetLocalSource(); src != nil {
				// Update request to match source connection
				request := &local_networkservice.NetworkServiceRequest{
					Connection:           src,
					MechanismPreferences: []*local_connection.Mechanism{src.GetMechanism()},
				}
				clientConnection.Request = request

				if dst := xcon.GetRemoteDestination(); dst != nil {
					srv.Heal(clientConnection, nsm.HealState_DstNmgrDown)
				}
				if dst := xcon.GetLocalDestination(); dst != nil {
					// In this case if there is no NSE, we just need to close.
					if endpoint != nil {
						srv.Heal(clientConnection, nsm.HealState_DstNmgrDown)
					} else {
						srv.closeLocalMissingNSE(clientConnection)
					}
				}
			}
			if src := xcon.GetLocalSource(); src != nil {
				if src.State == local_connection.State_DOWN {
					// if source is down, we need to close connection properly.
					_ = srv.Close(context.Background(), clientConnection)
				}
			}
			logrus.Infof("Active connection state %v is Restored", xcon)
		}
	}
	logrus.Infof("All connections are recovered...")
	// Notify state is restored
	srv.stateRestored <- true
}

func (srv *networkServiceManager) closeLocalMissingNSE(clientConnection *model.ClientConnection) {
	logrus.Infof("Local endopoint is not available, so closing local NSE connection %v", clientConnection)
	err := srv.close(context.Background(), clientConnection, true, true)
	if err != nil {
		logrus.Errorf("Failed to close local NSE(missing) connection %v", err)
	}
}

func (srv *networkServiceManager) RemoteConnectionLost(clientConnection nsm.NSMClientConnection) {
	connection := clientConnection.(*model.ClientConnection)
	connection.ConnectionState = model.ClientConnection_Healing
	logrus.Infof("NSM: Remote opened connection is not monitored and put into Healing state %v", clientConnection)
	go func() {
		<-time.Tick(srv.properties.HealTimeout)

		if connection.ConnectionState == model.ClientConnection_Healing {
			logrus.Errorf("NSM: Timeout happened for checking connection status from Healing.. %v. Closing connection...", clientConnection)
			// Nobody was healed connection from Remote side.
			if err := srv.Close(context.Background(), clientConnection); err != nil {
				logrus.Errorf("NSM: Error closing connection %v", err)
			}
		}
	}()
}

func (srv *networkServiceManager) closeEndpoint(ctx context.Context, clientConnection *model.ClientConnection) error {
	if clientConnection.Endpoint == nil {
		logrus.Infof("No need to close, since NSE is we know is dead at this point.")
		return nil
	}
	closeCtx, closeCancel := context.WithTimeout(ctx, srv.properties.CloseTimeout)
	defer closeCancel()

	client, nseClientError := srv.nseManager.createNSEClient(closeCtx, clientConnection.Endpoint)

	if client != nil {
		if ld := clientConnection.Xcon.GetLocalDestination(); ld != nil {
			return client.Close(ctx, ld)
		}
		if rd := clientConnection.Xcon.GetRemoteDestination(); rd != nil {
			return client.Close(ctx, rd)
		}
		err := client.Cleanup()
		if err != nil {
			logrus.Errorf("NSM: Error during Cleanup: %v", err)
		}
	} else {
		logrus.Errorf("NSM: Failed to create NSE Client %v", nseClientError)
	}
	return nseClientError
}
