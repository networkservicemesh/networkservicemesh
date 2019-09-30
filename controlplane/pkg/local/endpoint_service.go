// Copyright (c) 2019 Cisco and/or its affiliates.
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

package local

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/spanhelper"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	unified_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	unified_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
	plugin_api "github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	unified_nsm "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
)

// ConnectionService makes basic Mechanism selection for the incoming connection
type endpointService struct {
	nseManager     unified_nsm.NetworkServiceEndpointManager
	properties     *nsm.Properties
	pluginRegistry plugins.PluginRegistry
	model          model.Model
}

func (cce *endpointService) closeEndpoint(ctx context.Context, cc *model.ClientConnection) error {

	span := spanhelper.FromContext(ctx, "closeEndpoint")
	defer span.Finish()
	ctx = span.Context()
	logger := span.Logger()

	if cc.Endpoint == nil {
		logger.Infof("No need to close, since NSE is we know is dead at this point.")
		return nil
	}
	closeCtx, closeCancel := context.WithTimeout(ctx, cce.properties.CloseTimeout)
	defer closeCancel()

	client, nseClientError := cce.nseManager.CreateNSEClient(closeCtx, cc.Endpoint)

	if client != nil {
		if ld := cc.Xcon.GetLocalDestination(); ld != nil {
			return client.Close(ctx, ld)
		}
		if rd := cc.Xcon.GetRemoteDestination(); rd != nil {
			return client.Close(ctx, rd)
		}
		err := client.Cleanup()
		span.LogError(err)
	} else {
		span.LogError(nseClientError)
	}
	return nseClientError
}

func (cce *endpointService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logger := common.Log(ctx)
	clientConnection := common.ModelConnection(ctx)
	dp := common.Dataplane(ctx)
	endpoint := common.Endpoint(ctx)

	if clientConnection == nil {
		return nil, fmt.Errorf("client connection need to be passed")
	}
	client, err := cce.nseManager.CreateNSEClient(ctx, endpoint)
	if err != nil {
		// 7.2.6.1
		return nil, fmt.Errorf("NSM:(7.2.6.1) Failed to create NSE Client. %v", err)
	}
	defer func() {
		if cleanupErr := client.Cleanup(); cleanupErr != nil {
			logger.Errorf("NSM:(7.2.6.2) Error during Cleanup: %v", cleanupErr)
		}
	}()

	var message unified_networkservice.Request
	if cce.nseManager.IsLocalEndpoint(endpoint) {
		message = cce.createLocalNSERequest(endpoint, dp, request.Connection, clientConnection)
	} else {
		message = cce.createRemoteNSMRequest(endpoint, request.Connection, dp, clientConnection)
	}
	logger.Infof("NSM:(7.2.6.2) Requesting NSE with request %v", message)

	span := spanhelper.FromContext(ctx, "nse.request")
	ctx = span.Context()
	defer span.Finish()
	span.LogObject("nse.request", message)

	nseConn, e := client.Request(ctx, message)
	span.LogObject("nse.response", nseConn)
	if e != nil {
		e = fmt.Errorf("NSM:(7.2.6.2.1) error requesting networkservice from %+v with message %#v error: %s", endpoint, message, e)
		span.LogError(e)
		return nil, e
	}
	// 7.2.6.2.2
	if err = cce.updateConnectionContext(ctx, request.GetConnection(), nseConn); err != nil {
		err = fmt.Errorf("NSM:(7.2.6.2.2) failure Validating NSE Connection: %s", err)
		span.LogError(err)
		return nil, err
	}

	// 7.2.6.2.3 update connection parameters, add workspace if local nse
	cce.updateConnectionParameters(nseConn, endpoint)

	ctx = common.WithEndpointConnection(ctx, nseConn)

	return ProcessNext(ctx, request)
}

func (cce *endpointService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {

	clientConnection := common.ModelConnection(ctx)
	if clientConnection != nil {
		if err := cce.closeEndpoint(ctx, clientConnection); err != nil {
			return &empty.Empty{}, err
		}
	}

	return ProcessClose(ctx, connection)
}

func (cce *endpointService) createLocalNSERequest(endpoint *registry.NSERegistration, dp *model.Dataplane, requestConn *connection.Connection, clientConnection *model.ClientConnection) unified_networkservice.Request {
	// We need to obtain parameters for local mechanism
	localM := append([]unified_connection.Mechanism{}, dp.LocalMechanisms...)

	if clientConnection.ConnectionState == model.ClientConnectionHealing && endpoint == clientConnection.Endpoint {
		if localDst := clientConnection.Xcon.GetLocalDestination(); localDst != nil {
			return networkservice.NewRequest(
				&connection.Connection{
					Id:             localDst.GetId(),
					NetworkService: localDst.NetworkService,
					Context:        localDst.GetContext(),
					Labels:         localDst.GetLabels(),
				},
				localM,
			)
		}
	}

	return networkservice.NewRequest(
		&connection.Connection{
			Id:             cce.model.ConnectionID(), // ID for NSE is managed by NSMgr
			NetworkService: endpoint.GetNetworkService().GetName(),
			Context:        requestConn.GetContext(),
			Labels:         requestConn.GetLabels(),
		},
		localM,
	)
}

func (cce *endpointService) createRemoteNSMRequest(endpoint *registry.NSERegistration, requestConn *connection.Connection, dp *model.Dataplane, clientConnection *model.ClientConnection) unified_networkservice.Request {
	// We need to obtain parameters for remote mechanism
	remoteM := append([]unified_connection.Mechanism{}, dp.RemoteMechanisms...)

	// Try Heal only if endpoint are same as for existing connection.
	if clientConnection.ConnectionState == model.ClientConnectionHealing && endpoint == clientConnection.Endpoint {
		if remoteDst := clientConnection.Xcon.GetRemoteDestination(); remoteDst != nil {
			return remote_networkservice.NewRequest(
				&remote_connection.Connection{
					Id:                                   remoteDst.GetId(),
					NetworkService:                       remoteDst.NetworkService,
					Context:                              remoteDst.GetContext(),
					Labels:                               remoteDst.GetLabels(),
					DestinationNetworkServiceManagerName: endpoint.GetNetworkServiceManager().GetName(),
					SourceNetworkServiceManagerName:      cce.model.GetNsm().GetName(),
					NetworkServiceEndpointName:           endpoint.GetNetworkServiceEndpoint().GetName(),
				},
				remoteM,
			)
		}
	}

	return remote_networkservice.NewRequest(
		&remote_connection.Connection{
			Id:                                   "-",
			NetworkService:                       requestConn.GetNetworkService(),
			Context:                              requestConn.GetContext(),
			Labels:                               requestConn.GetLabels(),
			DestinationNetworkServiceManagerName: endpoint.GetNetworkServiceManager().GetName(),
			SourceNetworkServiceManagerName:      cce.model.GetNsm().GetName(),
			NetworkServiceEndpointName:           endpoint.GetNetworkServiceEndpoint().GetName(),
		},
		remoteM,
	)
}

func (cce *endpointService) validateConnection(ctx context.Context, conn unified_connection.Connection) error {
	if err := conn.IsComplete(); err != nil {
		return err
	}

	wrapper := plugin_api.NewConnectionWrapper(conn)
	result, err := cce.pluginRegistry.GetConnectionPluginManager().ValidateConnection(ctx, wrapper)
	if err != nil {
		return err
	}

	if result.GetStatus() != plugin_api.ConnectionValidationStatus_SUCCESS {
		return fmt.Errorf(result.GetErrorMessage())
	}

	return nil
}

func (cce *endpointService) updateConnectionContext(ctx context.Context, source *connection.Connection, destination unified_connection.Connection) error {
	if err := cce.validateConnection(ctx, destination); err != nil {
		return err
	}

	if err := source.UpdateContext(destination.GetContext()); err != nil {
		return err
	}

	return nil
}

func (cce *endpointService) updateConnectionParameters(nseConn unified_connection.Connection, endpoint *registry.NSERegistration) {
	if cce.nseManager.IsLocalEndpoint(endpoint) {
		modelEp := cce.model.GetEndpoint(endpoint.GetNetworkServiceEndpoint().GetName())
		if modelEp != nil { // In case of tests this could be empty
			nseConn.GetConnectionMechanism().GetParameters()[connection.Workspace] = modelEp.Workspace
			nseConn.GetConnectionMechanism().GetParameters()[connection.WorkspaceNSEName] = modelEp.Endpoint.GetNetworkServiceEndpoint().GetName()
		}
		logrus.Infof("NSM:(7.2.6.2.4) Update Local NSE connection parameters: %v", nseConn.GetConnectionMechanism())
	}
}

// NewEndpointService -  creates a service to connect to endpoint
func NewEndpointService(nseManager unified_nsm.NetworkServiceEndpointManager, properties *nsm.Properties, mdl model.Model, pluginRegistry plugins.PluginRegistry) networkservice.NetworkServiceServer {
	return &endpointService{
		nseManager:     nseManager,
		properties:     properties,
		model:          mdl,
		pluginRegistry: pluginRegistry,
	}
}
