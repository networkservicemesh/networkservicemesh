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

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	unified_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

const (
	// DataplaneRetryCount - A number of times to call Dataplane Request, TODO: Remove after DP will be stable.
	DataplaneRetryCount = 10
	// DataplaneRetryDelay - a delay between operations.
	DataplaneRetryDelay = 500 * time.Millisecond
	// DataplaneTimeout - A dataplane timeout
	DataplaneTimeout = 15 * time.Second
	// ErrorCloseTimeout - timeout to close all stuff in case of error
	ErrorCloseTimeout = 15 * time.Second
)

// dataplaneService -
type dataplaneService struct {
	serviceRegistry serviceregistry.ServiceRegistry
	model           model.Model
}

func (cce *dataplaneService) selectDataplane(request *networkservice.NetworkServiceRequest) (*model.Dataplane, error) {
	dp, err := cce.model.SelectDataplane(func(dp *model.Dataplane) bool {
		for _, m := range request.GetRequestMechanismPreferences() {
			if cce.findMechanism(dp.RemoteMechanisms, m.GetMechanismType()) != nil {
				return true
			}
		}
		return false
	})
	return dp, err
}
func (cce *dataplaneService) findMechanism(mechanismPreferences []unified_connection.Mechanism, mechanismType unified_connection.MechanismType) unified_connection.Mechanism {
	for _, m := range mechanismPreferences {
		if m.GetMechanismType() == mechanismType {
			return m
		}
	}
	return nil
}

func (cce *dataplaneService) selectRemoteMechanism(request *networkservice.NetworkServiceRequest, dp *model.Dataplane) (*connection.Mechanism, error) {
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

			vni := cce.serviceRegistry.VniAllocator().Vni(dpParameters[connection.VXLANSrcIP], parameters[connection.VXLANSrcIP])
			parameters[connection.VXLANVNI] = strconv.FormatUint(uint64(vni), 10)
		}
		return mechanism.(*connection.Mechanism), nil
	}

	return nil, fmt.Errorf("failed to select mechanism, no matched mechanisms found")
}

func (cce *dataplaneService) updateMechanism(request *networkservice.NetworkServiceRequest, dp *model.Dataplane) error {
	connection := request.GetConnection()
	// 5.x
	if m, err := cce.selectRemoteMechanism(request, dp); err == nil {
		connection.SetConnectionMechanism(m.Clone())
	} else {
		return err
	}

	if connection.GetConnectionMechanism() == nil {
		return fmt.Errorf("required mechanism are not found... %v ", request.GetRequestMechanismPreferences())
	}

	if connection.GetConnectionMechanism().GetParameters() == nil {
		connection.GetConnectionMechanism().SetParameters(map[string]string{})
	}

	return nil
}

func (cce *dataplaneService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logger := common.Log(ctx)

	clientConnection := common.ModelConnection(ctx)
	// 3. get dataplane
	err := cce.serviceRegistry.WaitForDataplaneAvailable(ctx, cce.model, DataplaneTimeout)
	if err != nil {
		logger.Errorf("Error waiting for dataplane: %v", err)
	}

	var dp *model.Dataplane

	// TODO: We could iterate dataplanes to match required one, if failed with first one.
	dp, err = cce.selectDataplane(request)
	if err != nil {
		return nil, err
	}

	// 5. Select a local dataplane and put it into conn object
	err = cce.updateMechanism(request, dp)
	if err != nil {
		// 5.1 Close Datplane connection, if had existing one and NSE is closed.
		cce.doFailureClose(ctx, request.GetConnection())
		return nil, fmt.Errorf("NSM:(5.1) %v", err)
	}

	logger.Infof("NSM:(5.1) Remote mechanism selected %v", request.Connection.Mechanism)

	ctx = common.WithDataplane(ctx, dp)
	conn, connErr := ProcessNext(ctx, request)
	if connErr != nil {
		return conn, connErr
	}
	// We need to programm dataplane.
	return cce.programmDataplane(ctx, conn, dp, clientConnection)
}

func (cce *dataplaneService) doFailureClose(ctx context.Context, conn *connection.Connection) {
	logger := common.Log(ctx)
	clientConnection := common.ModelConnection(ctx)

	newCtx, cancel := context.WithTimeout(context.Background(), ErrorCloseTimeout)
	defer cancel()
	newCtx = common.WithLog(newCtx, logger)
	newCtx = common.WithModelConnection(newCtx, clientConnection)

	_, closeErr := ProcessClose(ctx, conn)
	if closeErr != nil {
		logger.Errorf("Failed to close connection %v", closeErr)
	}
}

func (cce *dataplaneService) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	cc := common.ModelConnection(ctx)
	logger := common.Log(ctx)

	// Close endpoints, etc
	empty, err := ProcessClose(ctx, conn)
	if cc.DataplaneState != model.DataplaneStateNone {

		logger.Info("NSM.Dataplane: Closing cross connection on dataplane...")
		dp := cce.model.GetDataplane(cc.DataplaneRegisteredName)
		dataplaneClient, conn, err := cce.serviceRegistry.DataplaneConnection(ctx, dp)
		if err != nil {
			logger.Error(err)
			return empty, err
		}
		if conn != nil {
			defer func() { _ = conn.Close() }()
		}
		if _, err := dataplaneClient.Close(context.Background(), cc.Xcon); err != nil {
			logger.Error(err)
			return empty, err
		}
		logger.Info("NSM.Dataplane: Cross connection successfully closed on dataplane")
		cce.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
			cc.DataplaneState = model.DataplaneStateNone
		})
	}
	return empty, err
}

func (cce *dataplaneService) programmDataplane(ctx context.Context, conn *connection.Connection, dp *model.Dataplane, clientConnection *model.ClientConnection) (*connection.Connection, error) {
	logger := common.Log(ctx)
	dataplaneClient, dataplaneConn, err := cce.serviceRegistry.DataplaneConnection(ctx, dp)
	if err != nil {
		logger.Errorf("Error creating dataplane connection %v. Performing close", err)
		cce.doFailureClose(ctx, conn)
		return conn, err
	}
	if dataplaneConn != nil { // Required for testing
		defer func() {
			if closeErr := dataplaneConn.Close(); closeErr != nil {
				logger.Errorf("NSM:(6.1) Error during close Dataplane connection: %v", err)
			}
		}()
	}

	var newXcon *crossconnect.CrossConnect
	// 9. We need to program dataplane with our values.
	// 9.1 Sending updated request to dataplane.
	for dpRetry := 0; dpRetry < DataplaneRetryCount; dpRetry++ {
		if ctx.Err() != nil {
			cce.doFailureClose(ctx, conn)
			return nil, ctx.Err()
		}

		logger.Infof("NSM:(9.1) Sending request to dataplane: %v retry: %v", clientConnection.Xcon, dpRetry)
		dpCtx, cancel := context.WithTimeout(ctx, DataplaneTimeout)
		defer cancel()
		newXcon, err = dataplaneClient.Request(dpCtx, clientConnection.Xcon)
		if err != nil {
			logger.Errorf("NSM:(9.1.1) Dataplane request failed: %v retry: %v", err, dpRetry)

			// Let's try again with a short delay
			if dpRetry < DataplaneRetryCount-1 {
				<-time.After(DataplaneRetryDelay)
				continue
			}
			logger.Errorf("NSM:(9.1.2) Dataplane request  all retry attempts failed: %v", clientConnection.Xcon)
			// 9.3 If datplane configuration are failed, we need to close remore NSE actually.
			cce.doFailureClose(ctx, conn)
			return conn, err
		}

		// In case of context deadline, we need to close NSE and dataplane.
		if ctx.Err() != nil {
			cce.doFailureClose(context.Background(), conn)
			return nil, ctx.Err()
		}

		clientConnection.Xcon = newXcon

		logger.Infof("NSM:(9.2) Dataplane configuration successful %v", clientConnection.Xcon)
		break
	}

	return cce.updateClientConnection(ctx, conn, clientConnection, dp)
}

func (cce *dataplaneService) updateClientConnection(ctx context.Context, conn *connection.Connection, clientConnection *model.ClientConnection, dp *model.Dataplane) (*connection.Connection, error) {
	// Update connection context if it updated from dataplane.
	err := conn.UpdateContext(clientConnection.GetConnectionSource().GetContext())
	if err != nil {
		cce.doFailureClose(ctx, conn)
		return nil, err
	}

	clientConnection.DataplaneRegisteredName = dp.RegisteredName
	clientConnection.DataplaneState = model.DataplaneStateReady

	return conn, nil
}

// NewDataplaneService -  creates a service to program dataplane.
func NewDataplaneService(model model.Model, serviceRegistry serviceregistry.ServiceRegistry) networkservice.NetworkServiceServer {
	return &dataplaneService{
		model:           model,
		serviceRegistry: serviceRegistry,
	}
}
