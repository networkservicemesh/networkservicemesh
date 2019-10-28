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
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	unifiedconnection "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

const (
	// ForwarderRetryCount - A number of times to call Forwarder Request, TODO: Remove after DP will be stable.
	ForwarderRetryCount = 10
	// ForwarderRetryDelay - a delay between operations.
	ForwarderRetryDelay = 500 * time.Millisecond
	// ForwarderTimeout - A forwarder timeout
	ForwarderTimeout = 15 * time.Second
	// ErrorCloseTimeout - timeout to close all stuff in case of error
	ErrorCloseTimeout = 15 * time.Second
)

// forwarderService -
type forwarderService struct {
	serviceRegistry serviceregistry.ServiceRegistry
	model           model.Model
}

func (cce *forwarderService) selectForwarder(request *networkservice.NetworkServiceRequest) (*model.Forwarder, error) {
	dp, err := cce.model.SelectForwarder(func(dp *model.Forwarder) bool {
		for _, m := range request.GetRequestMechanismPreferences() {
			if cce.findMechanism(dp.RemoteMechanisms, m.GetMechanismType()) != nil {
				return true
			}
		}
		return false
	})
	return dp, err
}
func (cce *forwarderService) findMechanism(mechanismPreferences []unifiedconnection.Mechanism, mechanismType unifiedconnection.MechanismType) unifiedconnection.Mechanism {
	for _, m := range mechanismPreferences {
		if m.GetMechanismType() == mechanismType {
			return m
		}
	}
	return nil
}

func (cce *forwarderService) selectRemoteMechanism(request *networkservice.NetworkServiceRequest, dp *model.Forwarder) (*connection.Mechanism, error) {
	for _, mechanism := range request.GetRequestMechanismPreferences() {
		dpMechanism := cce.findMechanism(dp.RemoteMechanisms, connection.MechanismType_VXLAN)
		if dpMechanism == nil {
			continue
		}

		// TODO: Add other mechanisms support

		if mechanism.GetMechanismType() == connection.MechanismType_VXLAN {
			parameters := mechanism.GetParameters()
			dpParameters := dpMechanism.GetParameters()

			parameters[connection.VXLANDstIP] = dpParameters[connection.VXLANSrcIP]
			var vni uint32

			extSrcIP := parameters[connection.VXLANSrcIP]
			extDstIP := dpParameters[connection.VXLANSrcIP]
			srcIP := parameters[connection.VXLANSrcIP]
			dstIP := dpParameters[connection.VXLANSrcIP]

			if ip, ok := parameters[connection.VXLANSrcOriginalIP]; ok {
				srcIP = ip
			}

			if ip, ok := parameters[connection.VXLANDstExternalIP]; ok {
				extDstIP = ip
			}

			if extDstIP != extSrcIP {
				vni = cce.serviceRegistry.VniAllocator().Vni(extDstIP, extSrcIP)
			} else {
				vni = cce.serviceRegistry.VniAllocator().Vni(dstIP, srcIP)
			}

			parameters[connection.VXLANVNI] = strconv.FormatUint(uint64(vni), 10)
		}

		logrus.Infof("NSM:(5.1) Remote mechanism selected %v", mechanism)
		return mechanism.(*connection.Mechanism), nil
	}

	return nil, errors.New("failed to select mechanism, no matched mechanisms found")
}

func (cce *forwarderService) updateMechanism(request *networkservice.NetworkServiceRequest, dp *model.Forwarder) error {
	conn := request.GetConnection()
	// 5.x
	if m, err := cce.selectRemoteMechanism(request, dp); err == nil {
		conn.SetConnectionMechanism(m.Clone())
	} else {
		return err
	}

	if conn.GetConnectionMechanism() == nil {
		return errors.Errorf("required mechanism are not found... %v ", request.GetRequestMechanismPreferences())
	}

	if conn.GetConnectionMechanism().GetParameters() == nil {
		conn.GetConnectionMechanism().SetParameters(map[string]string{})
	}

	return nil
}

func (cce *forwarderService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logger := common.Log(ctx)
	span := spanhelper.GetSpanHelper(ctx)

	clientConnection := common.ModelConnection(ctx)
	// 3. get forwarder
	if err := cce.serviceRegistry.WaitForForwarderAvailable(ctx, cce.model, ForwarderTimeout); err != nil {
		logger.Errorf("Error waiting for forwarder: %v", err)
		return nil, err
	}

	// TODO: We could iterate forwarders to match required one, if failed with first one.
	dp, err := cce.selectForwarder(request)
	if err != nil {
		return nil, err
	}

	// 5. Select a local forwarder and put it into conn object
	err = cce.updateMechanism(request, dp)
	if err != nil {
		// 5.1 Close Datplane connection, if had existing one and NSE is closed.
		cce.doFailureClose(ctx)
		return nil, errors.Errorf("NSM:(5.1) %v", err)
	}

	span.LogObject("dataplane", dp)

	ctx = common.WithForwarder(ctx, dp)
	conn, connErr := ProcessNext(ctx, request)
	if connErr != nil {
		cce.doFailureClose(ctx)
		return conn, connErr
	}
	// We need to program forwarder.
	return cce.programForwarder(ctx, conn, dp, clientConnection)
}

func (cce *forwarderService) doFailureClose(ctx context.Context) {
	clientConnection := common.ModelConnection(ctx)

	newCtx, cancel := context.WithTimeout(context.Background(), ErrorCloseTimeout)
	defer cancel()

	span := spanhelper.CopySpan(newCtx, spanhelper.GetSpanHelper(ctx), "doForwarderClose")
	defer span.Finish()

	newCtx = span.Context()

	newCtx = common.WithLog(newCtx, span.Logger())
	newCtx = common.WithModelConnection(newCtx, clientConnection)

	closeErr := cce.performClose(newCtx, clientConnection, span.Logger())
	span.LogError(closeErr)
}

func (cce *forwarderService) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {

	cc := common.ModelConnection(ctx)
	logger := common.Log(ctx)
	empt, err := ProcessClose(ctx, conn)
	if closeErr := cce.performClose(ctx, cc, logger); closeErr != nil {
		logger.Errorf("Failed to close: %v", closeErr)
	}
	return empt, err
}

func (cce *forwarderService) performClose(ctx context.Context, cc *model.ClientConnection, logger logrus.FieldLogger) error {
	// Close endpoints, etc
	if cc.ForwarderState != model.ForwarderStateNone {
		logger.Info("NSM.Forwarder: Closing cross connection on forwarder...")
		dp := cce.model.GetForwarder(cc.ForwarderRegisteredName)
		forwarderClient, conn, err := cce.serviceRegistry.ForwarderConnection(ctx, dp)
		if err != nil {
			logger.Error(err)
			return err
		}
		if conn != nil {
			defer func() { _ = conn.Close() }()
		}
		if _, err := forwarderClient.Close(ctx, cc.Xcon); err != nil {
			logger.Error(err)
			return err
		}
		logger.Info("NSM.Forwarder: Cross connection successfully closed on forwarder")
		cc.ForwarderState = model.ForwarderStateNone
	}
	return nil
}

func (cce *forwarderService) programForwarder(ctx context.Context, conn *connection.Connection, dp *model.Forwarder, clientConnection *model.ClientConnection) (*connection.Connection, error) {
	span := spanhelper.FromContext(ctx, "programForwarder")
	defer span.Finish()
	// We need to program forwarder.
	forwarderClient, forwarderConn, err := cce.serviceRegistry.ForwarderConnection(ctx, dp)
	if err != nil {
		span.Logger().Errorf("Error creating forwarder connection %v. Performing close", err)
		cce.doFailureClose(span.Context())
		return conn, err
	}
	if forwarderConn != nil { // Required for testing
		defer func() {
			if closeErr := forwarderConn.Close(); closeErr != nil {
				span.Logger().Errorf("NSM:(6.1) Error during close Forwarder connection: %v", closeErr)
			}
		}()
	}

	var newXcon *crossconnect.CrossConnect
	// 9. We need to program forwarder with our values.
	// 9.1 Sending updated request to forwarder.
	for dpRetry := 0; dpRetry < ForwarderRetryCount; dpRetry++ {
		if ctx.Err() != nil {
			cce.doFailureClose(ctx)
			return nil, ctx.Err()
		}

		attemptSpan := spanhelper.FromContext(span.Context(), fmt.Sprintf("ProgramAttempt-%v", dpRetry))
		defer attemptSpan.Finish()
		attemptSpan.LogObject("attempt", dpRetry)

		span.Logger().Infof("NSM:(9.1) Sending request to forwarder")
		attemptSpan.LogObject("request", clientConnection.Xcon)

		dpCtx, cancel := context.WithTimeout(attemptSpan.Context(), ForwarderTimeout)
		newXcon, err = forwarderClient.Request(dpCtx, clientConnection.Xcon)
		cancel()
		if err != nil {
			attemptSpan.Logger().Errorf("NSM:(9.1.1) Forwarder request failed: %v retry: %v", err, dpRetry)

			// Let's try again with a short delay
			if dpRetry < ForwarderRetryCount-1 {
				<-time.After(ForwarderRetryDelay)
				continue
			}
			attemptSpan.Logger().Errorf("NSM:(9.1.2) Forwarder request  all retry attempts failed: %v", clientConnection.Xcon)
			// 9.3 If datplane configuration are failed, we need to close remore NSE actually.
			cce.doFailureClose(attemptSpan.Context())
			attemptSpan.Finish()
			return conn, err
		}

		// In case of context deadline, we need to close NSE and forwarder.
		ctxErr := attemptSpan.Context().Err()
		if ctxErr != nil {
			attemptSpan.LogError(ctxErr)
			cce.doFailureClose(attemptSpan.Context())
			attemptSpan.Finish()
			return nil, ctxErr
		}

		clientConnection.Xcon = newXcon

		attemptSpan.Logger().Infof("NSM:(9.2) Forwarder configuration successful")
		attemptSpan.LogObject("crossConnection", clientConnection.Xcon)
		break
	}

	// Update connection context if it updated from forwarder.
	return cce.updateClientConnection(ctx, conn, clientConnection, dp)
}

func (cce *forwarderService) updateClientConnection(ctx context.Context, conn *connection.Connection, clientConnection *model.ClientConnection, dp *model.Forwarder) (*connection.Connection, error) {
	// Update connection context if it updated from forwarder.
	err := conn.UpdateContext(clientConnection.GetConnectionSource().GetContext())
	if err != nil {
		cce.doFailureClose(ctx)
		return nil, err
	}

	clientConnection.ForwarderRegisteredName = dp.RegisteredName
	clientConnection.ForwarderState = model.ForwarderStateReady

	return conn, nil
}

// NewForwarderService -  creates a service to program forwarder.
func NewForwarderService(model model.Model, serviceRegistry serviceregistry.ServiceRegistry) networkservice.NetworkServiceServer {
	return &forwarderService{
		model:           model,
		serviceRegistry: serviceRegistry,
	}
}
