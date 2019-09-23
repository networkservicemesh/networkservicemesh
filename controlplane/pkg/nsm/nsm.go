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

	"github.com/opentracing/opentracing-go"

	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/local"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
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
	NetworkServiceHealProcessor
	sync.RWMutex

	serviceRegistry  serviceregistry.ServiceRegistry
	pluginRegistry   plugins.PluginRegistry
	model            model.Model
	properties       *nsm.Properties
	stateRestored    chan bool
	renamedEndpoints map[string]string
	nseManager       nsm.NetworkServiceEndpointManager

	localService  local_networkservice.NetworkServiceServer
	remoteService remote_networkservice.NetworkServiceServer
}

func (srv *networkServiceManager) LocalManager() local_networkservice.NetworkServiceServer {
	return srv.localService
}

func (srv *networkServiceManager) RemoteManager() remote_networkservice.NetworkServiceServer {
	return srv.remoteService
}

func (srv *networkServiceManager) SetRemoteServer(server remote_networkservice.NetworkServiceServer) {
	srv.remoteService = server
}

func (srv *networkServiceManager) ServiceRegistry() serviceregistry.ServiceRegistry {
	return srv.serviceRegistry
}
func (srv *networkServiceManager) NseManager() nsm.NetworkServiceEndpointManager {
	return srv.nseManager
}
func (srv *networkServiceManager) PluginRegistry() plugins.PluginRegistry {
	return srv.pluginRegistry
}
func (srv *networkServiceManager) Model() model.Model {
	return srv.model
}

func (srv *networkServiceManager) GetHealProperties() *nsm.Properties {
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

	healService := local.NewCompositeService(
		local.NewRequestValidator(),
		local.NewConnectionService(model),
		local.NewDataplaneService(model, serviceRegistry),
		local.NewEndpointSelectorService(nseManager, pluginRegistry),
		local.NewEndpointService(nseManager, properties, model, pluginRegistry),
		local.NewCrossConnectService(),
	)

	srv := &networkServiceManager{
		serviceRegistry:  serviceRegistry,
		pluginRegistry:   pluginRegistry,
		model:            model,
		properties:       properties,
		stateRestored:    make(chan bool, 1),
		renamedEndpoints: make(map[string]string),
		nseManager:       nseManager,
		localService:     healService,
	}

	srv.NetworkServiceHealProcessor = newNetworkServiceHealProcessor(
		serviceRegistry,
		model,
		properties,
		srv,
		nseManager,
	)

	return srv
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

func (srv *networkServiceManager) getNetworkServiceManagerName() string {
	return srv.model.GetNsm().GetName()
}

func (srv *networkServiceManager) WaitForDataplane(ctx context.Context, timeout time.Duration) error {
	// Wait for at least one dataplane to be available
	if err := srv.serviceRegistry.WaitForDataplaneAvailable(ctx, srv.model, timeout); err != nil {
		return err
	}
	logrus.Infof("Dataplane is available, waiting for initial state received and processed...")
	select {
	case <-srv.stateRestored:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("Failed to wait for NSMD stare restore... timeout %v happened", timeout)
	}
}

func (srv *networkServiceManager) RestoreConnections(xcons []*crossconnect.CrossConnect, dataplane string) {
	ctx := context.Background()
	var span opentracing.Span
	if opentracing.IsGlobalTracerRegistered() {
		span = opentracing.StartSpan("NSMgr.RestoreConnections")
		ctx = opentracing.ContextWithSpan(ctx, span)
		defer span.Finish()
	}

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
						logrus.Errorf("Error retrieving SRC/DST IP or VNI from Remote connection %v %v", err, err2)
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
						if xcon.GetRemoteDestination() != nil && ep.GetName() == xcon.GetRemoteDestination().GetNetworkServiceEndpointName() {
							endpoint = &registry.NSERegistration{
								NetworkServiceManager:  endpoints.NetworkServiceManagers[ep.NetworkServiceManagerName],
								NetworkServiceEndpoint: ep,
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
						err := srv.CloseConnection(ctx, clientConnection)
						if err != nil {
							logrus.Errorf("Failed to close local NSE connection %v", err)
						}
					}
					srv.RemoteConnectionLost(ctx, clientConnection)
				} else {
					srv.closeLocalMissingNSE(ctx, clientConnection)
				}
			} else {
				if dst := xcon.GetRemoteDestination(); dst != nil {
					srv.Heal(clientConnection, nsm.HealStateDstNmgrDown)
				} else {
					// In this case if there is no NSE, we just need to close.
					if endpoint != nil {
						srv.Heal(clientConnection, nsm.HealStateDstNmgrDown)
					} else {
						srv.closeLocalMissingNSE(ctx, clientConnection)
					}
				}

				if src.GetConnectionState() == connection.StateDown {
					// if source is down, we need to close connection properly.
					_ = srv.CloseConnection(ctx, clientConnection)
				}
			}
			logrus.Infof("Active connection state %v is Restored", xcon)
		}
	}
	logrus.Infof("All connections are recovered...")
	// Notify state is restored
	srv.stateRestored <- true
}

func (srv *networkServiceManager) closeLocalMissingNSE(ctx context.Context, cc nsm.ClientConnection) {
	logrus.Infof("Local endpoint is not available, so closing local NSE connection %v", cc)
	err := srv.CloseConnection(ctx, cc)
	if err != nil {
		logrus.Errorf("Failed to close local NSE(missing) connection %v", err)
	}
}

func (srv *networkServiceManager) RemoteConnectionLost(ctx context.Context, clientConnection nsm.ClientConnection) {
	logrus.Infof("NSM: Remote opened connection is not monitored and put into Healing state %v", clientConnection)

	srv.model.ApplyClientConnectionChanges(clientConnection.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionHealingBegin
	})

	go func() {
		<-time.After(srv.properties.HealTimeout)

		if modelCC := srv.model.GetClientConnection(clientConnection.GetID()); modelCC != nil && modelCC.ConnectionState == model.ClientConnectionHealing {
			logrus.Errorf("NSM: Timeout happened for checking connection status from Healing.. %v. Closing connection...", clientConnection)
			// Nobody was healed connection from Remote side.
			if err := srv.CloseConnection(ctx, clientConnection); err != nil {
				logrus.Errorf("NSM: Error closing connection %v", err)
			}
		}
	}()
}

func (srv *networkServiceManager) NotifyRenamedEndpoint(nseOldName, nseNewName string) {
	logrus.Infof("Notified about renamed endpoint %v => %v", nseOldName, nseNewName)
	srv.renamedEndpoints[nseOldName] = nseNewName
}
