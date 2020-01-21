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

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/properties"

	"github.com/pkg/errors"

	mechanismCommon "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

// ConnectionService makes basic Mechanism selection for the incoming connection
type endpointService struct {
	nseManager nsm.NetworkServiceEndpointManager
	props      *properties.Properties
	model      model.Model
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
	closeCtx, closeCancel := context.WithTimeout(ctx, cce.props.CloseTimeout)
	defer closeCancel()

	client, nseClientError := cce.nseManager.CreateNSEClient(closeCtx, cc.Endpoint)

	if client != nil {
		if ld := cc.Xcon.GetLocalDestination(); ld != nil {
			return client.Close(ctx, ld)
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

	return common.ProcessNext(ctx, request)
}

func (cce *endpointService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	clientConnection := common.ModelConnection(ctx)
	if clientConnection != nil {
		if err := cce.closeEndpoint(ctx, clientConnection); err != nil {
			return &empty.Empty{}, err
		}
	}

	return common.ProcessClose(ctx, connection)
}

func (cce *endpointService) createLocalNSERequest(endpoint *registry.NSERegistration, dp *model.Forwarder, requestConn *connection.Connection, clientConnection *model.ClientConnection) *networkservice.NetworkServiceRequest {
	// We need to obtain parameters for local mechanism
	localM := append([]*connection.Mechanism{}, dp.LocalMechanisms...)

	if clientConnection.ConnectionState == model.ClientConnectionHealing && endpoint == clientConnection.Endpoint {
		if localDst := clientConnection.Xcon.GetLocalDestination(); localDst != nil {
			return &networkservice.NetworkServiceRequest{
				Connection: &connection.Connection{
					Id:             localDst.GetId(),
					NetworkService: localDst.NetworkService,
					Context:        localDst.GetContext(),
					Labels:         localDst.GetLabels(),
					Path:           common.Strings2Path(cce.model.GetNsm().GetName()),
				},
				MechanismPreferences: localM,
			}
		}
	}

	return &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id:             cce.model.ConnectionID(), //NSMgr assign ID for local Endpoint connections
			NetworkService: endpoint.GetNetworkService().GetName(),
			Path:           common.Strings2Path(cce.model.GetNsm().GetName()),
			Context:        requestConn.GetContext(),
			Labels:         requestConn.GetLabels(),
		},
		MechanismPreferences: localM,
	}
}

func (cce *endpointService) validateConnection(ctx context.Context, conn *connection.Connection) error {
	if err := conn.IsComplete(); err != nil {
		return err
	}

	return nil
}

func (cce *endpointService) updateConnectionContext(ctx context.Context, source, destination *connection.Connection) error {
	if err := cce.validateConnection(ctx, destination); err != nil {
		return err
	}

	if err := source.UpdateContext(destination.GetContext()); err != nil {
		return err
	}

	return nil
}

func (cce *endpointService) updateConnectionParameters(nseConn *connection.Connection, endpoint *registry.NSERegistration) {
	if cce.nseManager.IsLocalEndpoint(endpoint) {
		modelEp := cce.model.GetEndpoint(endpoint.GetNetworkServiceEndpoint().GetName())
		if modelEp != nil { // In case of tests this could be empty
			nseConn.Mechanism.GetParameters()[mechanismCommon.Workspace] = modelEp.Workspace
			nseConn.Mechanism.GetParameters()[kernel.WorkspaceNSEName] = modelEp.Endpoint.GetNetworkServiceEndpoint().GetName()
		}
		logrus.Infof("NSM:(7.2.6.2.4) Update Local NSE connection parameters: %v", nseConn.Mechanism)
	}
}

// NewEndpointService -  creates a service to connect to endpoint
func NewEndpointService(nseManager nsm.NetworkServiceEndpointManager, properties *properties.Properties, mdl model.Model) networkservice.NetworkServiceServer {
	return &endpointService{
		nseManager: nseManager,
		props:      properties,
		model:      mdl,
	}
}
