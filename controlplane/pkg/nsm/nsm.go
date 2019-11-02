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

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/properties"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/local"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"

	mechanismCommon "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

const (
	ForwarderRetryCount = 10 // A number of times to call Forwarder Request, TODO: Remove after DP will be stable.
	ForwarderRetryDelay = 500 * time.Millisecond
	ForwarderTimeout    = 15 * time.Second
)

// Network service manager to manage both local/remote NSE connections.
type networkServiceManager struct {
	nsm.NetworkServiceHealProcessor
	sync.RWMutex

	serviceRegistry  serviceregistry.ServiceRegistry
	pluginRegistry   plugins.PluginRegistry
	model            model.Model
	props            *properties.Properties
	stateRestored    chan bool
	renamedEndpoints map[string]string
	nseManager       nsm.NetworkServiceEndpointManager

	remoteService networkservice.NetworkServiceServer
	ctx           context.Context
}

func (srv *networkServiceManager) Context() context.Context {
	return srv.ctx
}

func (srv *networkServiceManager) LocalManager(clientConnection nsm.ClientConnection) networkservice.NetworkServiceServer {
	return common.NewCompositeService("LocalHeal",
		common.NewRequestValidator(),
		common.NewMonitorService(clientConnection.(*model.ClientConnection).Monitor),
		local.NewConnectionService(srv.model),
		local.NewForwarderService(srv.model, srv.serviceRegistry),
		local.NewEndpointSelectorService(srv.nseManager, srv.pluginRegistry),
		local.NewEndpointService(srv.nseManager, srv.props, srv.model, srv.pluginRegistry),
		common.NewCrossConnectService(),
	)
}

func (srv *networkServiceManager) RemoteManager() networkservice.NetworkServiceServer {
	return srv.remoteService
}

func (srv *networkServiceManager) SetRemoteServer(server networkservice.NetworkServiceServer) {
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

func (srv *networkServiceManager) GetHealProperties() *properties.Properties {
	return srv.props
}

// NewNetworkServiceManager creates an instance of NetworkServiceManager
func NewNetworkServiceManager(ctx context.Context, model model.Model, serviceRegistry serviceregistry.ServiceRegistry, pluginRegistry plugins.PluginRegistry) nsm.NetworkServiceManager {
	properties := properties.NewNsmProperties()
	nseManager := &nseManager{
		serviceRegistry: serviceRegistry,
		model:           model,
		props:           properties,
	}

	srv := &networkServiceManager{
		serviceRegistry:  serviceRegistry,
		pluginRegistry:   pluginRegistry,
		model:            model,
		props:            properties,
		stateRestored:    make(chan bool, 1),
		renamedEndpoints: make(map[string]string),
		nseManager:       nseManager,
		ctx:              ctx,
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

func (srv *networkServiceManager) WaitForForwarder(ctx context.Context, timeout time.Duration) error {
	// Wait for at least one forwarder to be available
	if err := srv.serviceRegistry.WaitForForwarderAvailable(ctx, srv.model, timeout); err != nil {
		return err
	}
	logrus.Infof("Forwarder is available, waiting for initial state received and processed...")
	select {
	case <-srv.stateRestored:
		return nil
	case <-time.After(timeout):
		return errors.Errorf("failed to wait for NSMD stare restore... timeout %v happened", timeout)
	}
}

func (srv *networkServiceManager) RestoreConnections(xcons []*crossconnect.CrossConnect, forwarder string, manager nsm.MonitorManager) {
	span := spanhelper.FromContext(srv.Context(), "Nsmgr.RestoreConnections")
	defer span.Finish()
	logger := span.Logger()
	for _, xcon := range xcons {
		srv.restoreXconnection(span.Context(), xcon, logger, forwarder, manager)
	}
	logger.Infof("All connections are recovered...")
	// Notify state is restored
	srv.stateRestored <- true
}

func (srv *networkServiceManager) restoreXconnection(ctx context.Context, xcon *crossconnect.CrossConnect, logger logrus.FieldLogger, forwarder string, manager nsm.MonitorManager) {
	// Model should increase its id counter to max of xcons restored from forwarder
	srv.model.CorrectIDGenerator(xcon.GetId())
	span := spanhelper.FromContext(ctx, "restoreXConnection")
	defer span.Finish()
	span.LogObject("forwarder", forwarder)
	span.LogObject("xcon", xcon)

	existing := srv.model.GetClientConnection(xcon.GetId())
	if existing == nil {
		span.Logger().Infof("Restoring state of active connection %v", xcon)

		endpointName := ""
		networkServiceName := ""
		var connectionState model.ClientConnectionState

		dp := srv.model.GetForwarder(forwarder)
		discovery, err := srv.serviceRegistry.DiscoveryClient(span.Context())
		span.LogError(err)
		if err != nil {
			span.LogError(errors.WithMessage(err, "failed to created discovery client"))
			return
		}
		connectionState, networkServiceName, endpointName = srv.getConnectionParameters(xcon, logger)

		endpoint, endpointRenamed := srv.findEndpoint(span.Context(), endpointName, networkServiceName, discovery, xcon, span)

		var request *networkservice.NetworkServiceRequest
		workspaceName := ""
		if src := xcon.GetSource(); src != nil && !src.IsRemote() {
			// Update request to match source connection
			request = &networkservice.NetworkServiceRequest{
				Connection: src,
				MechanismPreferences: []*connection.Mechanism{
					src.Mechanism,
				},
			}
			workspaceName = src.GetMechanism().GetParameters()[mechanismCommon.Workspace]
		}

		monitor := manager.LocalConnectionMonitor(workspaceName)
		clientConnection := srv.createConnection(xcon, request, endpoint, dp, connectionState, monitor)
		srv.model.AddClientConnection(span.Context(), clientConnection)

		if monitor == nil {
			span.LogError(errors.Errorf("failed to restore connection %v. Workspace could be found for %v. closing", xcon, workspaceName))
			srv.CloseConnection(span.Context(), clientConnection)
			return
		}

		srv.performHeal(span.Context(), xcon, endpoint, endpointRenamed, clientConnection, logger)
		span.LogObject("restored", xcon)
	}
}

func (srv *networkServiceManager) findEndpoint(ctx context.Context, endpointName string, networkServiceName string, discovery registry.NetworkServiceDiscoveryClient, xcon *crossconnect.CrossConnect, span spanhelper.SpanHelper) (*registry.NSERegistration, bool) {
	var endpoint *registry.NSERegistration
	endpointRenamed := false
	if endpointName != "" {
		endpoint = srv.getEndpoint(ctx, networkServiceName, endpointName, discovery, xcon)

		if endpoint == nil {
			// Check if endpoint was renamed
			if newEndpointName, ok := srv.renamedEndpoints[endpointName]; ok {
				span.Logger().Infof("Endpoint was renamed %v => %v", endpointName, newEndpointName)
				localEndpoint := srv.model.GetEndpoint(newEndpointName)
				if localEndpoint != nil {
					endpoint = localEndpoint.Endpoint
					endpointRenamed = true
				}
			} else {
				span.LogError(errors.Errorf("failed to find Endpoint %s", endpointName))
			}
		} else {
			span.LogObject("endpoint", endpoint)
		}
	}
	return endpoint, endpointRenamed
}

func (srv *networkServiceManager) performHeal(ctx context.Context, xcon *crossconnect.CrossConnect, endpoint *registry.NSERegistration, endpointRenamed bool, clientConnection nsm.ClientConnection, logger logrus.FieldLogger) {
	// Add healing timer, for connection to be healed from source side.
	if src := xcon.GetSource(); src != nil && src.IsRemote() {
		if endpoint != nil {
			if endpointRenamed {
				// close current connection and wait for a new one
				err := srv.CloseConnection(ctx, clientConnection)
				if err != nil {
					logger.Errorf("Failed to close local NSE connection %v", err)
				}
			}
			srv.RemoteConnectionLost(ctx, clientConnection)
		} else {
			srv.closeLocalMissingNSE(ctx, clientConnection)
		}
	} else {
		if dst := xcon.GetRemoteDestination(); dst != nil {
			srv.Heal(ctx, clientConnection, nsm.HealStateDstNmgrDown)
		} else {
			// In this case if there is no NSE, we just need to close.
			if endpoint != nil {
				srv.Heal(ctx, clientConnection, nsm.HealStateDstNmgrDown)
			} else {
				srv.closeLocalMissingNSE(ctx, clientConnection)
			}
		}

		if src.GetState() == connection.State_DOWN {
			// if source is down, we need to close connection properly.
			_ = srv.CloseConnection(ctx, clientConnection)
		}
	}
}

func (srv *networkServiceManager) createConnection(xcon *crossconnect.CrossConnect, request *networkservice.NetworkServiceRequest, endpoint *registry.NSERegistration, dp *model.Forwarder, state model.ClientConnectionState, monitor connectionmonitor.MonitorServer) *model.ClientConnection {
	return &model.ClientConnection{
		ConnectionID:            xcon.GetId(),
		Request:                 request,
		Xcon:                    xcon,
		Endpoint:                endpoint, // We do not have endpoint here.
		ForwarderRegisteredName: dp.RegisteredName,
		ConnectionState:         state,
		ForwarderState:          model.ForwarderStateReady, // It is configured already.
		Monitor:                 monitor,
	}
}

func (srv *networkServiceManager) getEndpoint(ctx context.Context, networkServiceName, endpointName string, discovery registry.NetworkServiceDiscoveryClient, xcon *crossconnect.CrossConnect) (endpoint *registry.NSERegistration) {
	span := spanhelper.FromContext(ctx, "getEndpoint")
	defer span.Finish()
	span.Logger().Infof("Discovering endpoint at registry Network service: %s endpoint: %s ", networkServiceName, endpointName)

	localEndpoint := srv.model.GetEndpoint(endpointName)
	if localEndpoint != nil {
		endpoint = localEndpoint.Endpoint
		span.LogObject("endpoint", endpoint)
	} else {
		endpoints, err := discovery.FindNetworkService(span.Context(), &registry.FindNetworkServiceRequest{
			NetworkServiceName: networkServiceName,
		})
		span.LogError(err)
		for _, ep := range endpoints.NetworkServiceEndpoints {
			if xcon.GetRemoteDestination() != nil && ep.GetName() == xcon.GetRemoteDestination().GetNetworkServiceEndpointName() {
				endpoint = &registry.NSERegistration{
					NetworkServiceManager:  endpoints.NetworkServiceManagers[ep.NetworkServiceManagerName],
					NetworkServiceEndpoint: ep,
					NetworkService:         endpoints.NetworkService,
				}
				span.LogObject("endpoint", endpoint)
				break
			}
		}
	}
	return endpoint
}

func (srv *networkServiceManager) getConnectionParameters(xcon *crossconnect.CrossConnect, logger logrus.FieldLogger) (connectionState model.ClientConnectionState, networkServiceName, endpointName string) {
	connectionState = model.ClientConnectionReady
	if src := xcon.GetSource(); src != nil && src.IsRemote() {
		// Since source is remote, connection need to be healed.
		connectionState = model.ClientConnectionBroken

		networkServiceName = src.GetNetworkService()
		endpointName = src.GetNetworkServiceEndpointName()
	} else if dst := xcon.GetDestination(); dst != nil && !dst.IsRemote() {
		// Local NSE, connection is Ready
		networkServiceName = dst.GetNetworkService()
		endpointName = dst.GetMechanism().GetParameters()[kernel.WorkspaceNSEName]
	} else {
		// NSE is remote one, and source is local one, we are ready.
		networkServiceName = xcon.GetDestination().GetNetworkService()
		endpointName = xcon.GetDestination().GetNetworkServiceEndpointName()

		// In case VxLan is used we need to correct vlanId id generator.
		mm := dst.Mechanism
		if mm.GetType() == vxlan.MECHANISM {
			m := vxlan.ToMechanism(mm)
			srcIP, err := m.SrcIP()
			dstIP, err2 := m.DstIP()
			vni, err3 := m.VNI()
			if err != nil || err2 != nil || err3 != nil {
				logger.Errorf("Error retrieving SRC/DST IP or VNI from Remote connection %v %v", err, err2)
			} else {
				srv.serviceRegistry.VniAllocator().Restore(srcIP, dstIP, vni)
			}
		}
	}
	return connectionState, networkServiceName, endpointName
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

	srv.model.ApplyClientConnectionChanges(ctx, clientConnection.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionHealingBegin
	})

	go func() {
		<-time.After(srv.props.HealTimeout)

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
