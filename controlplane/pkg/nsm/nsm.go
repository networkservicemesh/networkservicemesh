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
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

const (
	DataplaneRetryCount = 10 // A number of times to call Dataplane Request, TODO: Remove after DP will be stable.
	DataplaneRetryDelay = 500 * time.Millisecond
	DataplaneTimeout    = 15 * time.Second
)

// Network service manager to manage both local/remote NSE connections.
type networkServiceManager struct {
	networkServiceHealProcessor
	sync.RWMutex

	serviceRegistry  serviceregistry.ServiceRegistry
	pluginRegistry   plugins.PluginRegistry
	model            model.Model
	properties       *nsm.NsmProperties
	stateRestored    chan bool
	renamedEndpoints map[string]string
	nseManager       networkServiceEndpointManager
}

func (srv *networkServiceManager) GetHealProperties() *nsm.NsmProperties {
	return srv.properties
}

// NewNetworkServiceManager creates an instance of NetworkServiceManager
func NewNetworkServiceManager(model model.Model, serviceRegistry serviceregistry.ServiceRegistry, pluginRegistry plugins.PluginRegistry) nsm.NetworkServiceManager {
	properties := nsm.NewNsmProperties()
	nseManager := &nseManager{
		serviceRegistry: serviceRegistry,
		model:           model,
		properties:      properties,
	}

	srv := &networkServiceManager{
		serviceRegistry:  serviceRegistry,
		pluginRegistry:   pluginRegistry,
		model:            model,
		properties:       properties,
		stateRestored:    make(chan bool, 1),
		renamedEndpoints: make(map[string]string),
		nseManager:       nseManager,
	}

	srv.networkServiceHealProcessor = newNetworkServiceHealProcessor(
		serviceRegistry,
		model,
		properties,
		srv,
		nseManager,
	)

	return srv
}

func (srv *networkServiceManager) Request(ctx context.Context, request networkservice.Request) (connection.Connection, error) {
	// Check if we are recovering connection, by checking passed connection Id is known to us.
	return srv.request(ctx, request, srv.model.GetClientConnection(request.GetRequestConnection().GetId()))
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

func (srv *networkServiceManager) request(ctx context.Context, request networkservice.Request, existingCC *model.ClientConnection) (connection.Connection, error) {
	requestID := create_logid()
	logrus.Infof("NSM:(%v) request: %v", requestID, request)

	if existingCC != nil {
		logrus.Infof("NSM:(%v) Called with existing connection passed: %v", requestID, existingCC)

		if modelCC := srv.model.GetClientConnection(existingCC.GetID()); modelCC == nil {
			return nil, fmt.Errorf("trying to request not existing connection")
		} else if modelCC.ConnectionState != model.ClientConnectionReady && modelCC.ConnectionState != model.ClientConnectionHealing {
			return nil, fmt.Errorf("trying to request connection in bad state")
		}

		srv.model.ApplyClientConnectionChanges(existingCC.GetID(), func(modelCC *model.ClientConnection) {
			modelCC.ConnectionState = model.ClientConnectionRequesting
		})
	}

	// 0. Make sure its a valid request
	err := request.IsValid()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	// 1. Create a new connection object.
	conn := request.GetRequestConnection().Clone()

	// 2. Set connection id for new connections.
	// Every NSMD manage it's connections.
	if existingCC == nil {
		conn.SetID(srv.createConnectionId())
	} else {
		// 2.1 we have connection updata/heal no need for new connection id
		conn.SetID(existingCC.GetID())
	}

	// 3. get dataplane
	srv.serviceRegistry.WaitForDataplaneAvailable(srv.model, DataplaneTimeout)

	dp, err := srv.model.SelectDataplane(func(dp *model.Dataplane) bool {
		if request.IsRemote() {
			for _, m := range request.GetRequestMechanismPreferences() {
				if findMechanism(dp.RemoteMechanisms, m.GetMechanismType()) != nil {
					return true
				}
			}
		} else {
			for _, m := range request.GetRequestMechanismPreferences() {
				if findMechanism(dp.LocalMechanisms, m.GetMechanismType()) != nil {
					return true
				}
			}
		}
		return false
	})

	if err != nil {
		srv.requestFailed(requestID, existingCC, existingCC, false, false)
		return nil, err
	}

	// A flag if we heal to close Dataplane in case of no NSE is found or failed to establish new connection.
	closeDataplaneOnNSEFailed := false

	// 4. Check if Heal/Update if we need to ask remote NSM or it is a just local mechanism change requested.
	// true if we detect we need to request NSE to upgrade/update connection.
	requestNSEOnUpdate := false
	if existingCC != nil {
		// 4.1 New Network service is requested, we need to close current connection and do re-request of NSE.
		if conn.GetNetworkService() != existingCC.GetNetworkService() {
			requestNSEOnUpdate = true
			closeDataplaneOnNSEFailed = true
			// Network service is closing, we need to close remote NSM and re-programm local one.
			if err = srv.closeEndpoint(ctx, existingCC); err != nil {
				logrus.Errorf("NSM:(4.1-%v) Error during close of NSE during Request.Upgrade %v Existing connection: %v error %v", requestID, request, existingCC, err)
			}
		} else {
			// 4.2 Check if NSE is still required, if some more context requests are different.
			requestNSEOnUpdate = srv.checkNeedNSERequest(requestID, conn, existingCC, dp)
		}
	}

	// 5. Select a local dataplane and put it into conn object
	err = srv.updateMechanism(requestID, conn, request, dp)
	if err != nil {
		// 5.1 Close Datplane connection, if had existing one and NSE is closed.
		srv.requestFailed(requestID, existingCC, existingCC, false, closeDataplaneOnNSEFailed)
		return nil, fmt.Errorf("NSM:(5.1-%v) %v", requestID, err)
	}

	// 6. Prepare dataplane connection is fine.
	logrus.Infof("NSM:(6-%v) Preparing to program dataplane: %v...", requestID, dp)
	dataplaneClient, dataplaneConn, err := srv.serviceRegistry.DataplaneConnection(dp)
	if err != nil {
		srv.requestFailed(requestID, existingCC, existingCC, false, false)
		return nil, err
	}
	if dataplaneConn != nil { // Required for testing
		defer func() {
			err := dataplaneConn.Close()
			if err != nil {
				logrus.Errorf("NSM:(6.1-%v) Error during close Dataplane connection: %v", requestID, err)
			}
		}()
	}

	var cc = existingCC

	// 7. do a Request() on NSE and select it.
	if existingCC == nil || requestNSEOnUpdate {
		// 7.1 try find NSE and do a Request to it.
		cc, err = srv.findConnectNSE(ctx, requestID, conn, existingCC, dp)
		if err != nil {
			srv.requestFailed(requestID, existingCC, existingCC, false, true)
			return nil, err
		}
	} else {
		// 7.2 We do not need to access NSE, since all parameters are same.
		cc.GetConnectionSource().SetConnectionMechanism(conn.GetConnectionMechanism())
		cc.GetConnectionSource().SetConnectionState(connection.StateUp)

		// 7.3 Destination context probably has been changed, so we need to update source context.
		if err = srv.updateConnectionContext(cc.GetConnectionSource(), cc.GetConnectionDestination()); err != nil {
			err = fmt.Errorf("NSM:(7.3-%v) Failed to update source connection context: %v", requestID, err)
			srv.requestFailed(requestID, cc, existingCC, true, true)
			return nil, err
		}
	}

	// 7.4 replace currently existing clientConnection or create new if it is absent
	srv.model.UpdateClientConnection(cc)

	// 8. Remember original Request for Heal cases.
	cc = srv.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.Request = request
	})

	var newXcon *crossconnect.CrossConnect
	// 9. We need to programm dataplane with our values.
	// 9.1 Sending updated request to dataplane.
	for dpRetry := 0; dpRetry < DataplaneRetryCount; dpRetry++ {
		if err := ctx.Err(); err != nil {
			srv.requestFailed(requestID, cc, existingCC, true, false)
			return nil, ctx.Err()
		}

		logrus.Infof("NSM:(9.1-%v) Sending request to dataplane: %v retry: %v", requestID, cc.Xcon, dpRetry)
		dpCtx, cancel := context.WithTimeout(context.Background(), DataplaneTimeout)
		defer cancel()
		newXcon, err = dataplaneClient.Request(dpCtx, cc.Xcon)
		if err != nil {
			logrus.Errorf("NSM:(9.1.1-%v) Dataplane request failed: %v retry: %v", requestID, err, dpRetry)

			// Let's try again with a short delay
			if dpRetry < DataplaneRetryCount-1 {
				<-time.After(DataplaneRetryDelay)
				continue
			}
			logrus.Errorf("NSM:(9.1.2-%v) Dataplane request  all retry attempts failed: %v", requestID, cc.Xcon)
			// 9.3 If datplane configuration are failed, we need to close remore NSE actually.
			srv.requestFailed(requestID, cc, existingCC, true, false)
			return nil, err
		}

		// In case of context deadline, we need to close NSE and dataplane.
		if err := ctx.Err(); err != nil {
			srv.requestFailed(requestID, cc, existingCC, true, false)
			return nil, ctx.Err()
		}

		logrus.Infof("NSM:(9.2-%v) Dataplane configuration successful %v", requestID, cc.Xcon)
		break
	}

	// 10. Send update for client connection
	srv.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.ConnectionState = model.ClientConnectionReady
		cc.DataplaneState = model.DataplaneStateReady
		cc.Xcon = newXcon
	})

	// 11. We are done with configuration here.
	logrus.Infof("NSM:(11-%v) Request done...", requestID)

	return cc.GetConnectionSource(), nil
}

func (srv *networkServiceManager) requestFailed(requestID string, cc, existingCC *model.ClientConnection, closeNSE, closeDp bool) {
	logrus.Errorf("NSM:(%v) Request failed", requestID)

	if cc == nil {
		return
	}

	if closeNSE {
		if err := srv.closeEndpoint(context.Background(), cc); err != nil {
			logrus.Errorf("NSM:(%v) Error closing NSE: %v", requestID, err)
		}
	}

	if closeDp {
		if err := srv.closeDataplane(cc); err != nil {
			logrus.Errorf("NSM:(%v) Error closing dataplane: %v", requestID, err)
		}
	}

	if existingCC == nil {
		srv.model.DeleteClientConnection(cc.GetID())
	}

	srv.model.ApplyClientConnectionChanges(cc.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionBroken
	})
}

func (srv *networkServiceManager) findConnectNSE(ctx context.Context, requestID string, conn connection.Connection, existingCC *model.ClientConnection, dp *model.Dataplane) (*model.ClientConnection, error) {
	// 7.x
	var endpoint *registry.NSERegistration
	var err error
	var last_error error
	var cc *model.ClientConnection
	ignoreEndpoints := map[string]*registry.NSERegistration{}
	for {
		if err := ctx.Err(); err != nil {
			logrus.Errorf("NSM:(7.1.0-%v) Context timeout, during find/call NSE... %v", requestID, err)
			return nil, err
		}
		endpoint = nil
		// 7.1.1 Clone Connection to support iteration via EndPoints
		nseConn := conn.Clone()

		if existingCC != nil {
			// 7.1.2 Check previous endpoint, and it we will be able to contact it, it should be fine.
			var connectionID string
			if dst := existingCC.Xcon.GetRemoteDestination(); dst != nil {
				connectionID = dst.GetId()
			}
			if dst := existingCC.Xcon.GetLocalDestination(); dst != nil {
				connectionID = dst.GetId()
			}

			if connectionID != "-" && existingCC.Endpoint != nil && ignoreEndpoints[existingCC.Endpoint.NetworkserviceEndpoint.EndpointName] == nil {
				endpoint = existingCC.Endpoint
			}
		}
		// 7.1.3 Check if endpoint is not ignored yet

		if endpoint == nil {
			// 7.1.4 Choose a new endpoint
			endpoint, err = srv.nseManager.getEndpoint(ctx, nseConn, ignoreEndpoints)
		}
		if err != nil {
			// 7.1.5 No endpoints found, we need to return error, including last error for previous NSE
			if last_error != nil {
				return nil, fmt.Errorf("NSM:(7.1.5-%v) %v. Last NSE Error: %v", requestID, err, last_error)
			} else {
				return nil, err
			}
		}
		// 7.1.6 Update Request with exclude_prefixes, etc
		if err = srv.updateConnection(nseConn); err != nil {
			return nil, fmt.Errorf("NSM:(7.1.6-%v) Failed to update connection: %v", requestID, err)
		}

		// 7.1.7 perform request to NSE/remote NSMD/NSE
		cc, err = srv.performNSERequest(ctx, requestID, endpoint, nseConn, dp, existingCC)

		// 7.1.8 in case of error we put NSE into ignored list to check another one.
		if err != nil {
			logrus.Errorf("NSM:(7.1.8-%v) NSE respond with error: %v ", requestID, err)
			last_error = err
			ignoreEndpoints[endpoint.NetworkserviceEndpoint.EndpointName] = endpoint
			continue
		}
		// 7.1.9 We are fine with NSE connection and could continue.
		return cc, nil
	}
}

func (srv *networkServiceManager) Close(ctx context.Context, clientConnection nsm.ClientConnection) error {
	cc := clientConnection.(*model.ClientConnection)

	if modelCC := srv.model.GetClientConnection(cc.GetID()); modelCC == nil || modelCC.ConnectionState == model.ClientConnectionClosing {
		return fmt.Errorf("closing already closed connection")
	}

	srv.model.ApplyClientConnectionChanges(cc.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionClosing
	})

	logrus.Infof("NSM: Closing connection %v", cc)

	nseErr := srv.closeEndpoint(ctx, cc)
	dpErr := srv.closeDataplane(cc)

	// TODO: We need to be sure Dataplane is respond well so we could delete connection.
	srv.model.DeleteClientConnection(cc.GetID())

	if nseErr != nil || dpErr != nil {
		return fmt.Errorf("NSM: Close error: %v", []error{nseErr, dpErr})
	}

	return nil
}

func (srv *networkServiceManager) performNSERequest(ctx context.Context, requestID string, endpoint *registry.NSERegistration, requestConn connection.Connection, dp *model.Dataplane, existingCC *model.ClientConnection) (*model.ClientConnection, error) {
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
			logrus.Errorf("NSM:(7.2.6.2-%v) Error during Cleanup: %v", requestID, err)
		}
	}()

	var message networkservice.Request
	if srv.nseManager.isLocalEndpoint(endpoint) {
		message = srv.createLocalNSERequest(endpoint, dp, requestConn)
	} else {
		message = srv.createRemoteNSMRequest(endpoint, requestConn, dp, existingCC)
	}
	logrus.Infof("NSM:(7.2.6.2-%v) Requesting NSE with request %v", requestID, message)
	nseConn, e := client.Request(ctx, message)

	if e != nil {
		logrus.Errorf("NSM:(7.2.6.2.1-%v) error requesting networkservice from %+v with message %#v error: %s", requestID, endpoint, message, e)
		return nil, e
	}

	// 7.2.6.2.2
	if err = srv.updateConnectionContext(requestConn, nseConn); err != nil {
		err = fmt.Errorf("NSM:(7.2.6.2.2-%v) failure Validating NSE Connection: %s", requestID, err)
		return nil, err
	}

	// 7.2.6.2.3 update connection parameters, add workspace if local nse
	srv.updateConnectionParameters(requestID, nseConn, endpoint)

	// 7.2.6.2.4 create cross connection
	dpAPIConnection := srv.createCrossConnect(requestConn, nseConn, endpoint)
	var dpState model.DataplaneState
	if existingCC != nil {
		dpState = existingCC.DataplaneState
	}
	clientConnection := &model.ClientConnection{
		ConnectionID:            requestConn.GetId(),
		Xcon:                    dpAPIConnection,
		Endpoint:                endpoint,
		DataplaneRegisteredName: dp.RegisteredName,
		ConnectionState:         model.ClientConnectionRequesting,
		DataplaneState:          dpState,
	}
	// 7.2.6.2.5 - It not a local NSE put remote NSM name in request
	if !srv.nseManager.isLocalEndpoint(endpoint) {
		clientConnection.RemoteNsm = endpoint.GetNetworkServiceManager()
	}
	return clientConnection, nil
}

func (srv *networkServiceManager) createCrossConnect(requestConn, nseConn connection.Connection, endpoint *registry.NSERegistration) *crossconnect.CrossConnect {
	return crossconnect.NewCrossConnect(
		requestConn.GetId(),
		endpoint.GetNetworkService().GetPayload(),
		requestConn,
		nseConn,
	)
}

func (srv *networkServiceManager) createConnectionId() string {
	return srv.model.ConnectionID()
}

func (srv *networkServiceManager) closeDataplane(cc *model.ClientConnection) error {
	if cc.DataplaneState == model.DataplaneStateNone {
		// Do not need to close
		return nil
	}
	logrus.Info("NSM.Dataplane: Closing cross connection on dataplane...")
	dp := srv.model.GetDataplane(cc.DataplaneRegisteredName)
	dataplaneClient, conn, err := srv.serviceRegistry.DataplaneConnection(dp)
	if err != nil {
		logrus.Error(err)
		return err
	}
	if conn != nil {
		defer conn.Close()
	}
	if _, err := dataplaneClient.Close(context.Background(), cc.Xcon); err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Info("NSM.Dataplane: Cross connection successfully closed on dataplane")
	srv.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.DataplaneState = model.DataplaneStateNone
	})

	return nil
}

func (srv *networkServiceManager) getNetworkServiceManagerName() string {
	return srv.model.GetNsm().GetName()
}

func (srv *networkServiceManager) updateConnection(conn connection.Connection) error {
	if conn.GetContext() == nil {
		c := &connectioncontext.ConnectionContext{}
		conn.SetContext(c)
	}

	return srv.pluginRegistry.GetConnectionPluginManager().UpdateConnection(conn)
}

func (srv *networkServiceManager) updateConnectionContext(source, destination connection.Connection) error {
	if err := srv.validateConnection(destination); err != nil {
		return err
	}

	if err := source.UpdateContext(destination.GetContext()); err != nil {
		return err
	}

	return nil
}

func (srv *networkServiceManager) validateConnection(conn connection.Connection) error {
	if err := conn.IsComplete(); err != nil {
		return err
	}

	if err := srv.pluginRegistry.GetConnectionPluginManager().ValidateConnection(conn); err != nil {
		return err
	}

	return nil
}

func (srv *networkServiceManager) updateConnectionParameters(requestID string, nseConn connection.Connection, endpoint *registry.NSERegistration) {
	if srv.nseManager.isLocalEndpoint(endpoint) {
		modelEp := srv.model.GetEndpoint(endpoint.GetNetworkserviceEndpoint().GetEndpointName())
		if modelEp != nil { // In case of tests this could be empty
			nseConn.GetConnectionMechanism().GetParameters()[local_connection.Workspace] = modelEp.Workspace
			nseConn.GetConnectionMechanism().GetParameters()[local_connection.WorkspaceNSEName] = modelEp.Endpoint.NetworkserviceEndpoint.EndpointName
		}
		logrus.Infof("NSM:(7.2.6.2.4-%v) Update Local NSE connection parameters: %v", requestID, nseConn.GetConnectionMechanism())
	}
}

/**
check if we need to do a NSE/Remote NSM request in case of our connection Upgrade/Healing procedure.
*/
func (srv *networkServiceManager) checkNeedNSERequest(requestID string, nsmConn connection.Connection, existingCC *model.ClientConnection, dp *model.Dataplane) bool {
	// 4.2.x
	// 4.2.1 Check if context is changed, if changed we need to
	if !proto.Equal(nsmConn.GetContext(), existingCC.GetConnectionSource().GetContext()) {
		return true
	}
	// We need to check, dp has mechanism changes in our Remote connection selected mechanism.

	if dst := existingCC.GetConnectionDestination(); dst.IsRemote() {
		dstM := dst.GetConnectionMechanism()

		// 4.2.2 Let's check if remote destination is matchs our dataplane destination.
		if dpM := findMechanism(dp.RemoteMechanisms, dstM.GetMechanismType()); dpM != nil {
			// 4.2.3 We need to check if source mechanism type and source parameters are different
			for k, v := range dpM.GetParameters() {
				rmV := dstM.GetParameters()[k]
				if v != rmV {
					logrus.Infof("NSM:(4.2.3-%v) Remote mechanism parameter %s was different with previous one : %v  %v", requestID, k, rmV, v)
					return true
				}
			}
			if !dpM.Equals(dstM) {
				logrus.Infof("NSM:(4.2.4-%v)  Remote mechanism was different with previous selected one : %v  %v", requestID, dstM, dpM)
				return true
			}
		} else {
			logrus.Infof("NSM:(4.2.5-%v) Remote mechanism previously selected was not found: %v  in dataplane %v", requestID, dstM, dp.RemoteMechanisms)
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
	case <-time.After(timeout):
		return fmt.Errorf("Failed to wait for NSMD stare restore... timeout %v happened", timeout)
	}
}

func (srv *networkServiceManager) RestoreConnections(xcons []*crossconnect.CrossConnect, dataplane string) {
	for _, xcon := range xcons {

		// Model should increase its id counter to max of xcons restored from dataplane
		srv.model.CorrectIDGenerator(xcon.GetId())

		existing := srv.model.GetClientConnection(xcon.GetId())
		if existing == nil {
			logrus.Infof("Restoring state of active connection %v", xcon)

			endpointName := ""
			networkServiceName := ""
			var endpoint *registry.NSERegistration
			connectionState := model.ClientConnectionReady

			dp := srv.model.GetDataplane(dataplane)

			discovery, err := srv.serviceRegistry.DiscoveryClient()
			if err != nil {
				logrus.Errorf("Failed to find NSE to recovery: %v", err)
			}

			if src := xcon.GetSourceConnection(); src.IsRemote() {
				// Since source is remote, connection need to be healed.
				connectionState = model.ClientConnectionBroken

				networkServiceName = src.GetNetworkService()
				endpointName = src.GetNetworkServiceEndpointName()
			} else if dst := xcon.GetDestinationConnection(); !dst.IsRemote() {
				// Local NSE, connection is Ready
				connectionState = model.ClientConnectionReady

				networkServiceName = dst.GetNetworkService()
				endpointName = dst.GetConnectionMechanism().GetParameters()[local_connection.WorkspaceNSEName]
			} else {
				// NSE is remote one, and source is local one, we are ready.
				connectionState = model.ClientConnectionReady

				networkServiceName = xcon.GetRemoteDestination().GetNetworkService()
				endpointName = xcon.GetRemoteDestination().GetNetworkServiceEndpointName()

				// In case VxLan is used we need to correct vlanId id generator.
				m := dst.GetConnectionMechanism().(*remote_connection.Mechanism)
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

			endpointRenamed := false
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
					// Check if endpoint was renamed
					if newEndpointName, ok := srv.renamedEndpoints[endpointName]; ok {
						logrus.Infof("Endpoint was renamed %v => %v", endpointName, newEndpointName)
						localEndpoint = srv.model.GetEndpoint(newEndpointName)
						if localEndpoint != nil {
							endpoint = localEndpoint.Endpoint
							endpointRenamed = true
						}
					} else {
						logrus.Errorf("Failed to find Endpoint %s", endpointName)
					}
				} else {
					logrus.Infof("Endpoint found: %v", endpoint)
				}
			}

			var request networkservice.Request
			if src := xcon.GetSourceConnection(); !src.IsRemote() {
				// Update request to match source connection
				request = local_networkservice.NewRequest(
					src,
					[]connection.Mechanism{src.GetConnectionMechanism()},
				)
			}

			clientConnection := &model.ClientConnection{
				ConnectionID:            xcon.GetId(),
				Request:                 request,
				Xcon:                    xcon,
				Endpoint:                endpoint, // We do not have endpoint here.
				DataplaneRegisteredName: dp.RegisteredName,
				ConnectionState:         connectionState,
				DataplaneState:          model.DataplaneStateReady, // It is configured already.
			}

			srv.model.AddClientConnection(clientConnection)

			// Add healing timer, for connection to be healed from source side.
			if src := xcon.GetSourceConnection(); src.IsRemote() {
				if endpoint != nil {
					if endpointRenamed {
						// close current connection and wait for a new one
						err := srv.Close(context.Background(), clientConnection)
						if err != nil {
							logrus.Errorf("Failed to close local NSE connection %v", err)
						}
					}
					srv.RemoteConnectionLost(clientConnection)
				} else {
					srv.closeLocalMissingNSE(clientConnection)
				}
			} else {
				if dst := xcon.GetRemoteDestination(); dst != nil {
					srv.Heal(clientConnection, nsm.HealStateDstNmgrDown)
				} else {
					// In this case if there is no NSE, we just need to close.
					if endpoint != nil {
						srv.Heal(clientConnection, nsm.HealStateDstNmgrDown)
					} else {
						srv.closeLocalMissingNSE(clientConnection)
					}
				}

				if src.GetConnectionState() == connection.StateDown {
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

func (srv *networkServiceManager) closeLocalMissingNSE(cc *model.ClientConnection) {
	logrus.Infof("Local endopoint is not available, so closing local NSE connection %v", cc)
	err := srv.Close(context.Background(), cc)
	if err != nil {
		logrus.Errorf("Failed to close local NSE(missing) connection %v", err)
	}
}

func (srv *networkServiceManager) RemoteConnectionLost(clientConnection nsm.ClientConnection) {
	logrus.Infof("NSM: Remote opened connection is not monitored and put into Healing state %v", clientConnection)

	srv.model.ApplyClientConnectionChanges(clientConnection.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionHealing
	})

	go func() {
		<-time.After(srv.properties.HealTimeout)

		if modelCC := srv.model.GetClientConnection(clientConnection.GetID()); modelCC != nil && modelCC.ConnectionState == model.ClientConnectionHealing {
			logrus.Errorf("NSM: Timeout happened for checking connection status from Healing.. %v. Closing connection...", clientConnection)
			// Nobody was healed connection from Remote side.
			if err := srv.Close(context.Background(), clientConnection); err != nil {
				logrus.Errorf("NSM: Error closing connection %v", err)
			}
		}
	}()
}

func (srv *networkServiceManager) NotifyRenamedEndpoint(nseOldName, nseNewName string) {
	logrus.Infof("Notified about renamed endpoint %v => %v", nseOldName, nseNewName)
	srv.renamedEndpoints[nseOldName] = nseNewName
}

func (srv *networkServiceManager) closeEndpoint(ctx context.Context, cc *model.ClientConnection) error {
	if cc.Endpoint == nil {
		logrus.Infof("No need to close, since NSE is we know is dead at this point.")
		return nil
	}
	closeCtx, closeCancel := context.WithTimeout(ctx, srv.properties.CloseTimeout)
	defer closeCancel()

	client, nseClientError := srv.nseManager.createNSEClient(closeCtx, cc.Endpoint)

	if client != nil {
		if ld := cc.Xcon.GetLocalDestination(); ld != nil {
			return client.Close(ctx, ld)
		}
		if rd := cc.Xcon.GetRemoteDestination(); rd != nil {
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
