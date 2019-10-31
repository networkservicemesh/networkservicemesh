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

package remote

import (
	"context"

	unified_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/sdk/compat"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	localconnection "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	unifiedconnection "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	unifiednetworkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
	pluginapi "github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	unifiednsm "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
)

// ConnectionService makes basic Mechanism selection for the incoming connection
type endpointService struct {
	nseManager     unifiednsm.NetworkServiceEndpointManager
	properties     *nsm.Properties
	pluginRegistry plugins.PluginRegistry
	model          model.Model
}

func (cce *endpointService) closeEndpoint(ctx context.Context, cc *model.ClientConnection) error {
	span := spanhelper.FromContext(ctx, "closeEndpoint")
	defer span.Finish()
	ctx = span.Context()
	logger := span.Logger()
	ctx = common.WithLog(ctx, logger)

	if cc.Endpoint == nil {
		logger.Infof("No need to close, since NSE is we know is dead at this point.")
		return nil
	}
	closeCtx, closeCancel := context.WithTimeout(ctx, cce.properties.CloseTimeout)
	defer closeCancel()

	client, nseClientError := cce.nseManager.CreateNSEClient(closeCtx, cc.Endpoint)

	if client != nil {
		if ld := cc.Xcon.GetLocalDestination(); ld != nil {
			return client.Close(ctx, compat.ConnectionUnifiedToNSM(ld))
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
	dp := common.Forwarder(ctx)
	endpoint := common.Endpoint(ctx)

	if clientConnection == nil {
		return nil, errors.Errorf("client connection need to be passed")
	}
	client, err := cce.nseManager.CreateNSEClient(ctx, endpoint)
	if err != nil {
		// 7.2.6.1
		return nil, errors.Errorf("NSM:(7.2.6.1) Failed to create NSE Client. %v", err)
	}
	defer func() {
		if cleanupErr := client.Cleanup(); cleanupErr != nil {
			logger.Errorf("NSM:(7.2.6.2) Error during Cleanup: %v", cleanupErr)
		}
	}()

	message := cce.createLocalNSERequest(endpoint, dp, request.Connection, clientConnection)
	logger.Infof("NSM:(7.2.6.2) Requesting NSE with request %v", message)

	span := spanhelper.FromContext(ctx, "nse.request")
	defer span.Finish()
	ctx = span.Context()
	span.LogObject("nse.request", message)

	nseConn, e := client.Request(ctx, message)
	span.LogObject("nse.response", nseConn)
	if e != nil {
		e = errors.Errorf("NSM:(7.2.6.2.1) error requesting networkservice from %+v with message %#v error: %s", endpoint, message, e)
		span.LogError(e)
		return nil, e
	}

	// 7.2.6.2.2
	if err = cce.updateConnectionContext(ctx, request.GetConnection(), nseConn); err != nil {
		err = errors.Errorf("NSM:(7.2.6.2.2) failure Validating NSE Connection: %s", err)
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

func (cce *endpointService) createLocalNSERequest(endpoint *registry.NSERegistration, dp *model.Forwarder, requestConn *connection.Connection, clientConnection *model.ClientConnection) unifiednetworkservice.Request {
	// We need to obtain parameters for local mechanism
	localM := append([]*unified.Mechanism{}, dp.LocalMechanisms...)

	if clientConnection.ConnectionState == model.ClientConnectionHealing && endpoint == clientConnection.Endpoint {
		if localDst := clientConnection.Xcon.GetLocalDestination(); localDst != nil {
			return compat.NetworkServiceRequestUnifiedToLocal(&unified_networkservice.NetworkServiceRequest{
				Connection: &unified.Connection{
					Id:             localDst.GetId(),
					NetworkService: localDst.NetworkService,
					Context:        localDst.GetContext(),
					Labels:         localDst.GetLabels(),
				},
				MechanismPreferences: localM,
			})
		}
	}

	return compat.NetworkServiceRequestUnifiedToLocal(&unified_networkservice.NetworkServiceRequest{
		Connection: &unified.Connection{
			Id:             cce.model.ConnectionID(), //TODO: NSE should assign this ID.
			NetworkService: endpoint.GetNetworkService().GetName(),
			Context:        requestConn.GetContext(),
			Labels:         requestConn.GetLabels(),
		},
		MechanismPreferences: localM,
	})
}

func (cce *endpointService) createRemoteNSMRequest(endpoint *registry.NSERegistration, requestConn *connection.Connection, dp *model.Forwarder, clientConnection *model.ClientConnection) unifiednetworkservice.Request {
	// We need to obtain parameters for remote mechanism
	remoteM := append([]*unified.Mechanism{}, dp.RemoteMechanisms...)

	// Try Heal only if endpoint are same as for existing connection.
	if clientConnection.ConnectionState == model.ClientConnectionHealing && endpoint == clientConnection.Endpoint {
		if remoteDst := clientConnection.Xcon.GetRemoteDestination(); remoteDst != nil {
			return compat.NetworkServiceRequestUnifiedToLocal(&unified_networkservice.NetworkServiceRequest{
				Connection: &unified.Connection{
					Id:             remoteDst.GetId(),
					NetworkService: remoteDst.NetworkService,
					Context:        remoteDst.GetContext(),
					Labels:         remoteDst.GetLabels(),
					NetworkServiceManagers: []string{
						cce.model.GetNsm().GetName(),                  // src
						endpoint.GetNetworkServiceManager().GetName(), // dst
					},
					NetworkServiceEndpointName: endpoint.GetNetworkServiceEndpoint().GetName(),
				},
				MechanismPreferences: remoteM,
			})
		}
	}

	return compat.NetworkServiceRequestUnifiedToLocal(&unified_networkservice.NetworkServiceRequest{
		Connection: &unified.Connection{
			Id:             cce.model.ConnectionID(),
			NetworkService: requestConn.GetNetworkService(),
			Context:        requestConn.GetContext(),
			Labels:         requestConn.GetLabels(),
			NetworkServiceManagers: []string{
				cce.model.GetNsm().GetName(),                  //src
				endpoint.GetNetworkServiceManager().GetName(), // dst
			},
			NetworkServiceEndpointName: endpoint.GetNetworkServiceEndpoint().GetName(),
		},
		MechanismPreferences: remoteM,
	})
}

func (cce *endpointService) validateConnection(ctx context.Context, conn unifiedconnection.Connection) error {
	if err := conn.IsComplete(); err != nil {
		return err
	}

	wrapper := pluginapi.NewConnectionWrapper(conn)
	result, err := cce.pluginRegistry.GetConnectionPluginManager().ValidateConnection(ctx, wrapper)
	if err != nil {
		return err
	}

	if result.GetStatus() != pluginapi.ConnectionValidationStatus_SUCCESS {
		return errors.Errorf(result.GetErrorMessage())
	}

	return nil
}

func (cce *endpointService) updateConnectionContext(ctx context.Context, source *connection.Connection, destination unifiedconnection.Connection) error {
	if err := cce.validateConnection(ctx, destination); err != nil {
		return err
	}

	if err := source.UpdateContext(destination.GetContext()); err != nil {
		return err
	}

	return nil
}

func (cce *endpointService) updateConnectionParameters(nseConn unifiedconnection.Connection, endpoint *registry.NSERegistration) {
	if cce.nseManager.IsLocalEndpoint(endpoint) {
		modelEp := cce.model.GetEndpoint(endpoint.GetNetworkServiceEndpoint().GetName())
		if modelEp != nil { // In case of tests this could be empty
			nseConn.GetConnectionMechanism().GetParameters()[localconnection.Workspace] = modelEp.Workspace
			nseConn.GetConnectionMechanism().GetParameters()[localconnection.WorkspaceNSEName] = modelEp.Endpoint.GetNetworkServiceEndpoint().GetName()
		}
		logrus.Infof("NSM:(7.2.6.2.4) Update Local NSE connection parameters: %v", nseConn.GetConnectionMechanism())
	}
}

// NewEndpointService -  creates a service to connect to endpoint
func NewEndpointService(nseManager unifiednsm.NetworkServiceEndpointManager, properties *nsm.Properties, mdl model.Model, pluginRegistry plugins.PluginRegistry) networkservice.NetworkServiceServer {
	return &endpointService{
		nseManager:     nseManager,
		properties:     properties,
		model:          mdl,
		pluginRegistry: pluginRegistry,
	}
}
