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
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	unified_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	plugin_api "github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	unified_nsm "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/plugins"
)

// ConnectionService makes basic Mechanism selection for the incoming connection
type endpointSelectorService struct {
	nseManager     unified_nsm.NetworkServiceEndpointManager
	pluginRegistry plugins.PluginRegistry
	model          model.Model
}

func (cce *endpointSelectorService) updateConnection(ctx context.Context, conn *connection.Connection) (*connection.Connection, error) {
	if conn.GetContext() == nil {
		c := &connectioncontext.ConnectionContext{}
		conn.SetContext(c)
	}

	wrapper := plugin_api.NewConnectionWrapper(conn)
	wrapper, err := cce.pluginRegistry.GetConnectionPluginManager().UpdateConnection(ctx, wrapper)
	if err != nil {
		return conn, err
	}

	return wrapper.GetConnection().(*connection.Connection), nil
}

func (cce *endpointSelectorService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logger := common.Log(ctx)
	clientConnection := common.ModelConnection(ctx)

	if clientConnection == nil {
		return nil, fmt.Errorf("client connection need to be passed")
	}

	// 4. Check if Heal/Update if we need to ask remote NSM or it is a just local mechanism change requested.
	// true if we detect we need to request NSE to upgrade/update connection.
	// 4.1 New Network service is requested, we need to close current connection and do re-request of NSE.
	requestNSEOnUpdate := cce.checkNSERequestIsRequired(ctx, clientConnection, request, logger)

	// 7. do a Request() on NSE and select it.
	if clientConnection.ConnectionState == model.ClientConnectionHealing && !requestNSEOnUpdate {
		return cce.checkUpdateConnectionContext(ctx, request, clientConnection)
	}

	// 7.1 try find NSE and do a Request to it.
	var endpoint *model.Endpoint

	// 7.1.1 Clone Connection to support iteration via EndPoints
	newRequest := request.Clone().(*networkservice.NetworkServiceRequest)
	nseConn := newRequest.Connection

	targetEndpoint := request.Connection.GetNetworkServiceEndpointName()
	if len(targetEndpoint) > 0 {
		endpoint = cce.model.GetEndpoint(targetEndpoint)
	}
	if endpoint == nil {
		return nil, fmt.Errorf("could not find endpoint with name: %s at local registry", targetEndpoint)
	}
	logger.Infof("selected endpoint %v", endpoint)
	// 7.1.6 Update Request with exclude_prefixes, etc
	nseConn, err := cce.updateConnection(ctx, nseConn)
	if err != nil {
		return nil, fmt.Errorf("error NSM:(7.1.6) Failed to update connection: %v", err)
	}

	// 7.1.7 perform request to NSE/remote NSMD/NSE
	ctx = common.WithEndpoint(ctx, endpoint.Endpoint)

	newRequest.Connection = nseConn
	conn, err := ProcessNext(ctx, newRequest)

	// 7.1.8 in case of error we put NSE into ignored list to check another one.
	if err != nil {
		logger.Errorf("NSM:(7.1.8) NSE respond with error: %v ", err)
		return nil, err
	}

	// We could put endpoint to clientConnection.
	clientConnection.Endpoint = endpoint.Endpoint
	// 7.1.9 We are fine with NSE connection and could continue.
	return conn, nil
}

func (cce *endpointSelectorService) checkNSERequestIsRequired(ctx context.Context, clientConnection *model.ClientConnection, request *networkservice.NetworkServiceRequest, logger logrus.FieldLogger) bool {
	requestNSEOnUpdate := false
	if clientConnection.ConnectionState == model.ClientConnectionHealing {
		if request.Connection.GetNetworkService() != clientConnection.GetNetworkService() ||
			request.Connection.GetNetworkServiceEndpointName() != clientConnection.Endpoint.GetNetworkServiceEndpoint().Name {
			requestNSEOnUpdate = true

			// Just close, since client connection already passed with context.
			// Network service is closing, we need to close remote NSM and re-program local one.
			if _, err := ProcessClose(ctx, request.GetConnection()); err != nil {
				logger.Errorf("NSM:(4.1) Error during close of NSE during Request.Upgrade %v Existing connection: %v error %v", request, clientConnection, err)
			}
		} else if !proto.Equal(request.Connection.GetContext(), clientConnection.GetConnectionSource().GetContext()) {
			// 4.2 Check if NSE is still required, if some more context requests are different.
			// 4.2.1 Check if context is changed, if changed we need to
			requestNSEOnUpdate = true
			logger.Infof("Context is different, NSE request is required")
		}
	}
	return requestNSEOnUpdate
}

func (cce *endpointSelectorService) validateConnection(ctx context.Context, conn unified_connection.Connection) error {
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

func (cce *endpointSelectorService) updateConnectionContext(ctx context.Context, source *connection.Connection, destination unified_connection.Connection) error {
	if err := cce.validateConnection(ctx, destination); err != nil {
		return err
	}

	if err := source.UpdateContext(destination.GetContext()); err != nil {
		return err
	}

	return nil
}

func (cce *endpointSelectorService) findMechanism(mechanismPreferences []unified_connection.Mechanism, mechanismType unified_connection.MechanismType) unified_connection.Mechanism {
	for _, m := range mechanismPreferences {
		if m.GetMechanismType() == mechanismType {
			return m
		}
	}
	return nil
}

func (cce *endpointSelectorService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	return ProcessClose(ctx, connection)
}

func (cce *endpointSelectorService) checkUpdateConnectionContext(ctx context.Context, request *networkservice.NetworkServiceRequest, clientConnection *model.ClientConnection) (*connection.Connection, error) {
	// We do not need to do request to endpoint and just need to update all stuff.
	// 7.2 We do not need to access NSE, since all parameters are same.
	logger := common.Log(ctx)
	clientConnection.GetConnectionSource().SetConnectionMechanism(request.Connection.GetConnectionMechanism())
	clientConnection.GetConnectionSource().SetConnectionState(unified_connection.StateUp)

	// 7.3 Destination context probably has been changed, so we need to update source context.
	if err := cce.updateConnectionContext(ctx, request.GetConnection(), clientConnection.GetConnectionDestination()); err != nil {
		err = fmt.Errorf("NSM:(7.3) Failed to update source connection context: %v", err)

		// Just close since client connection is already passed with context
		if _, closeErr := ProcessClose(ctx, request.GetConnection()); closeErr != nil {
			logger.Errorf("Failed to perform close: %v", closeErr)
		}
		return nil, err
	}
	return request.Connection, nil
}

// NewEndpointSelectorService -  creates a service to select remote endpoint.
func NewEndpointSelectorService(nseManager unified_nsm.NetworkServiceEndpointManager,
	pluginRegistry plugins.PluginRegistry, model model.Model) networkservice.NetworkServiceServer {
	return &endpointSelectorService{
		nseManager:     nseManager,
		pluginRegistry: pluginRegistry,
		model:          model,
	}
}
