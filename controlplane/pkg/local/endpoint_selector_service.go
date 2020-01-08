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

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
)

// ConnectionService makes basic Mechanism selection for the incoming connection
type endpointSelectorService struct {
	nseManager nsm.NetworkServiceEndpointManager
}

func (cce *endpointSelectorService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logger := common.Log(ctx)
	span := spanhelper.GetSpanHelper(ctx)
	clientConnection := common.ModelConnection(ctx)
	fwd := common.Forwarder(ctx)

	if clientConnection == nil {
		return nil, errors.Errorf("client connection need to be passed")
	}

	// Check if Heal/Update if we need to ask remote NSM or it is a just local mechanism change requested.
	// true if we detect we need to request NSE to upgrade/update connection.
	// New Network service is requested, we need to close current connection and do re-request of NSE.
	requestNSEOnUpdate := cce.checkNSEUpdateIsRequired(ctx, clientConnection, request, logger, fwd)
	span.LogObject("requestNSEOnUpdate", requestNSEOnUpdate)

	// Do a Request() on NSE and select it.
	if clientConnection.ConnectionState == model.ClientConnectionHealing && !requestNSEOnUpdate {
		return cce.checkUpdateConnectionContext(ctx, request, clientConnection)
	}

	// Try find NSE and do a Request to it.
	var lastError error
	ignoreEndpoints := common.IgnoredEndpoints(ctx)
	parentCtx := ctx
	attempt := 0
	for {
		attempt++
		span := spanhelper.FromContext(parentCtx, fmt.Sprintf("select-nse-%v", attempt))

		logger := span.Logger()
		ctx = common.WithLog(span.Context(), logger)

		// Clone Connection to support iteration via EndPoints
		newRequest, endpoint, err := cce.prepareRequest(ctx, request, clientConnection, ignoreEndpoints)
		if err != nil {
			span.Finish()
			return cce.combineErrors(span, lastError, err)
		}
		if err = cce.checkTimeout(parentCtx, span); err != nil {
			span.Finish()
			return nil, err
		}

		// Perform request to NSE/remote NSMD/NSE
		ctx = common.WithEndpoint(ctx, endpoint)
		// Perform passing execution to next chain element.
		conn, err := common.ProcessNext(ctx, newRequest)

		// In case of error we put NSE into ignored list to check another one.
		if err != nil {
			logger.Errorf("NSM: endpointSelectorService: NSE respond with error: %v ", err)
			lastError = err
			ignoreEndpoints[endpoint.GetEndpointNSMName()] = endpoint
			span.Finish()
			continue
		}
		// We could put endpoint to clientConnection.
		clientConnection.Endpoint = endpoint
		if !cce.nseManager.IsLocalEndpoint(endpoint) {
			clientConnection.RemoteNsm = endpoint.GetNetworkServiceManager()
		}
		// We are fine with NSE connection and could continue.
		span.Finish()
		return conn, nil
	}
}

func (cce *endpointSelectorService) combineErrors(span spanhelper.SpanHelper, err, lastError error) (*connection.Connection, error) {
	if lastError != nil {
		span.LogError(lastError)
		return nil, errors.Errorf("NSM: endpointSelectorService: %v. Last NSE Error: %v", err, lastError)
	}
	return nil, err
}

func (cce *endpointSelectorService) selectEndpoint(ctx context.Context, clientConnection *model.ClientConnection, ignoreEndpoints map[registry.EndpointNSMName]*registry.NSERegistration, nseConn *connection.Connection) (*registry.NSERegistration, error) {
	var endpoint *registry.NSERegistration
	if clientConnection.ConnectionState == model.ClientConnectionHealing {
		// Check previous endpoint, and it we will be able to contact it, it should be fine.
		endpointName := clientConnection.Endpoint.GetEndpointNSMName()
		if clientConnection.Endpoint != nil && ignoreEndpoints[endpointName] == nil {
			endpoint = clientConnection.Endpoint
		} else {
			// Ignored, we need to update DSTid.
			clientConnection.Xcon.Destination.Id = "-"
		}
		//TODO: Add check if endpoint are in registry or not.
	}
	// Check if endpoint is not ignored yet
	if endpoint == nil {
		// Choose a new endpoint
		return cce.nseManager.GetEndpoint(ctx, nseConn, ignoreEndpoints)
	}
	return endpoint, nil
}

func (cce *endpointSelectorService) checkNSEUpdateIsRequired(ctx context.Context, clientConnection *model.ClientConnection, request *networkservice.NetworkServiceRequest, logger logrus.FieldLogger, fwd *model.Forwarder) bool {
	requestNSEOnUpdate := false
	if clientConnection.ConnectionState == model.ClientConnectionHealing {
		if request.Connection.GetNetworkService() != clientConnection.GetNetworkService() {
			requestNSEOnUpdate = true

			// Just close, since client connection already passed with context.
			// Network service is closing, we need to close remote NSM and re-program local one.
			if _, err := common.ProcessClose(ctx, request.GetConnection()); err != nil {
				logger.Errorf("NSM: endpointSelectorService: Error during close of NSE during Request.Upgrade %v Existing connection: %v error %v", request, clientConnection, err)
			}
		} else {
			// Check if NSE is still required, if some more context requests are different.
			requestNSEOnUpdate = cce.checkNeedNSERequest(logger, request.Connection, clientConnection, fwd)
			if requestNSEOnUpdate {
				logger.Infof("Context is different, NSE request is required")
			}
		}
	}
	return requestNSEOnUpdate
}

func (cce *endpointSelectorService) validateConnection(ctx context.Context, conn *connection.Connection) error {
	return conn.IsComplete()
}

func (cce *endpointSelectorService) updateConnectionContext(ctx context.Context, source, destination *connection.Connection) error {
	if err := cce.validateConnection(ctx, destination); err != nil {
		return err
	}

	if err := source.UpdateContext(destination.GetContext()); err != nil {
		return err
	}

	return nil
}

/**
check if we need to do a NSE/Remote NSM request in case of our connection Upgrade/Healing procedure.
*/
func (cce *endpointSelectorService) checkNeedNSERequest(logger logrus.FieldLogger, nsmConn *connection.Connection, existingCC *model.ClientConnection, fwd *model.Forwarder) bool {
	// Check if context is changed, if changed we need to
	if !proto.Equal(nsmConn.GetContext(), existingCC.GetConnectionSource().GetContext()) {
		return true
	}
	// We need to check, forwarder has mechanism changes in our Remote connection selected mechanism.

	if dst := existingCC.GetConnectionDestination(); dst.IsRemote() {
		dstM := dst.GetMechanism()

		// Let's check if remote destination is matches our forwarder destination.
		if fwdM := cce.findMechanism(fwd.RemoteMechanisms, dstM.GetType()); fwdM != nil {
			// We need to check if source mechanism type and source parameters are different
			for k, v := range fwdM.GetParameters() {
				rmV := dstM.GetParameters()[k]
				if v != rmV {
					logger.Infof("NSM: endpointSelectorService: Remote mechanism parameter %s was different with previous one : %v  %v", k, rmV, v)
					return true
				}
			}
			if !fwdM.Equals(dstM) {
				logger.Infof("NSM: endpointSelectorService: Remote mechanism was different with previous selected one : %v  %v", dstM, fwdM)
				return true
			}
		} else {
			logger.Infof("NSM: endpointSelectorService: Remote mechanism previously selected was not found: %v  in forwarder %v", dstM, fwd.RemoteMechanisms)
			return true
		}
	}

	return false
}

func (cce *endpointSelectorService) findMechanism(mechanismPreferences []*connection.Mechanism, mechanismType string) *connection.Mechanism {
	for _, m := range mechanismPreferences {
		if m.GetType() == mechanismType {
			return m
		}
	}
	return nil
}

func (cce *endpointSelectorService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	return common.ProcessClose(ctx, connection)
}

func (cce *endpointSelectorService) checkUpdateConnectionContext(ctx context.Context, request *networkservice.NetworkServiceRequest, clientConnection *model.ClientConnection) (*connection.Connection, error) {
	// We do not need to do request to endpoint and just need to update all stuff.
	// We do not need to access NSE, since all parameters are same.
	logger := common.Log(ctx)
	clientConnection.Xcon.Source.Mechanism = request.Connection.GetMechanism()
	clientConnection.Xcon.Source.State = connection.State_UP

	// Destination context probably has been changed, so we need to update source context.
	if err := cce.updateConnectionContext(ctx, request.GetConnection(), clientConnection.GetConnectionDestination()); err != nil {
		err = errors.Errorf("NSM: endpointSelectorService failed to update source connection context: %v", err)

		// Just close since client connection is already passed with context
		if _, closeErr := common.ProcessClose(ctx, request.GetConnection()); closeErr != nil {
			logger.Errorf("Failed to perform close: %v", closeErr)
		}
		return nil, err
	}

	if !cce.nseManager.IsLocalEndpoint(clientConnection.Endpoint) {
		clientConnection.RemoteNsm = clientConnection.Endpoint.GetNetworkServiceManager()
	}
	return request.Connection, nil
}

func (cce *endpointSelectorService) prepareRequest(ctx context.Context, request *networkservice.NetworkServiceRequest, clientConnection *model.ClientConnection, ignoreEndpoints map[registry.EndpointNSMName]*registry.NSERegistration) (*networkservice.NetworkServiceRequest, *registry.NSERegistration, error) {
	newRequest := request.Clone()
	nseConn := newRequest.Connection
	span := spanhelper.GetSpanHelper(ctx)

	endpoint, err := cce.selectEndpoint(ctx, clientConnection, ignoreEndpoints, nseConn)
	if err != nil {
		return nil, nil, err
	}

	span.LogObject("selected endpoint", endpoint)
	if nseConn.GetContext() == nil {
		nseConn.Context = &connectioncontext.ConnectionContext{}
	}

	newRequest.Connection = nseConn
	return newRequest, endpoint, nil
}

func (cce *endpointSelectorService) checkTimeout(ctx context.Context, span spanhelper.SpanHelper) error {
	if ctx.Err() != nil {
		newErr := errors.Errorf("NSM: endpointSelectorService: Context timeout, during find/call NSE... %v", ctx.Err())
		span.LogError(newErr)
		return newErr
	}
	return nil
}

// NewEndpointSelectorService - creates a service to select endpoint
func NewEndpointSelectorService(nseManager nsm.NetworkServiceEndpointManager) networkservice.NetworkServiceServer {
	return &endpointSelectorService{
		nseManager: nseManager,
	}
}
