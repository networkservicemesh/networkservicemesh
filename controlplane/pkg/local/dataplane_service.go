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
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
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
			if cce.findMechanism(dp.LocalMechanisms, m.GetMechanismType()) != nil {
				return true
			}
		}
		return false
	})
	return dp, err
}
func (cce *dataplaneService) findMechanism(mechanismPreferences []unified.Mechanism, mechanismType unified.MechanismType) unified.Mechanism {
	for _, m := range mechanismPreferences {
		if m.GetMechanismType() == mechanismType {
			return m
		}
	}
	return nil
}

func (cce *dataplaneService) updateMechanism(request *networkservice.NetworkServiceRequest, dp *model.Dataplane) error {
	conn := request.GetConnection()
	// 5.x
	for _, m := range request.GetRequestMechanismPreferences() {
		if dpMechanism := cce.findMechanism(dp.LocalMechanisms, m.GetMechanismType()); dpMechanism != nil {
			conn.SetConnectionMechanism(m.Clone())
			break
		}
	}

	if conn.GetConnectionMechanism() == nil {
		return errors.Errorf("required mechanism are not found... %v ", request.GetRequestMechanismPreferences())
	}

	if conn.GetConnectionMechanism().GetParameters() == nil {
		conn.GetConnectionMechanism().SetParameters(map[string]string{})
	}

	return nil
}

func (cce *dataplaneService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logger := common.Log(ctx)

	clientConnection := common.ModelConnection(ctx)
	// 3. get dataplane
	if err := cce.serviceRegistry.WaitForDataplaneAvailable(ctx, cce.model, DataplaneTimeout); err != nil {
		logger.Errorf("Error waiting for dataplane: %v", err)
		return nil, err
	}

	// TODO: We could iterate dataplanes to match required one, if failed with first one.
	dp, err := cce.selectDataplane(request)
	if err != nil {
		return nil, err
	}

	// 5. Select a local dataplane and put it into conn object
	err = cce.updateMechanism(request, dp)
	if err != nil {
		return nil, errors.Errorf("NSM:(5.1) %v", err)
	}

	ctx = common.WithDataplane(ctx, dp)
	conn, connErr := ProcessNext(ctx, request)
	if connErr != nil {
		return conn, connErr
	}
	// We need to program dataplane.
	return cce.programDataplane(ctx, conn, dp, clientConnection)
}

func (cce *dataplaneService) doFailureClose(ctx context.Context) {
	clientConnection := common.ModelConnection(ctx)

	newCtx, cancel := context.WithTimeout(context.Background(), ErrorCloseTimeout)
	defer cancel()

	span := spanhelper.CopySpan(newCtx, spanhelper.GetSpanHelper(ctx), "doDataplaneClose")
	defer span.Finish()

	newCtx = span.Context()

	newCtx = common.WithLog(newCtx, span.Logger())
	newCtx = common.WithModelConnection(newCtx, clientConnection)

	closeErr := cce.performClose(newCtx, clientConnection, span.Logger())
	span.LogError(closeErr)
}

func (cce *dataplaneService) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	cc := common.ModelConnection(ctx)
	logger := common.Log(ctx)
	empt, err := ProcessClose(ctx, conn)
	if closeErr := cce.performClose(ctx, cc, logger); closeErr != nil {
		logger.Errorf("Failed to close: %v", closeErr)
	}
	return empt, err
}

func (cce *dataplaneService) performClose(ctx context.Context, cc *model.ClientConnection, logger logrus.FieldLogger) error {
	// Close endpoints, etc
	if cc.DataplaneState != model.DataplaneStateNone {
		logger.Info("NSM.Dataplane: Closing cross connection on dataplane...")
		dp := cce.model.GetDataplane(cc.DataplaneRegisteredName)
		dataplaneClient, conn, err := cce.serviceRegistry.DataplaneConnection(ctx, dp)
		if err != nil {
			logger.Error(err)
			return err
		}
		if conn != nil {
			defer func() { _ = conn.Close() }()
		}
		if _, err := dataplaneClient.Close(ctx, cc.Xcon); err != nil {
			logger.Error(err)
			return err
		}
		logger.Info("NSM.Dataplane: Cross connection successfully closed on dataplane")
		cc.DataplaneState = model.DataplaneStateNone
	}
	return nil
}

func (cce *dataplaneService) programDataplane(ctx context.Context, conn *connection.Connection, dp *model.Dataplane, clientConnection *model.ClientConnection) (*connection.Connection, error) {
	logger := common.Log(ctx)
	// We need to program dataplane.
	dataplaneClient, dataplaneConn, err := cce.serviceRegistry.DataplaneConnection(ctx, dp)
	if err != nil {
		logger.Errorf("Error creating dataplane connection %v. Performing close", err)
		cce.doFailureClose(ctx)
		return conn, err
	}
	if dataplaneConn != nil { // Required for testing
		defer func() {
			if closeErr := dataplaneConn.Close(); closeErr != nil {
				logger.Errorf("NSM:(6.1) Error during close Dataplane connection: %v", closeErr)
			}
		}()
	}

	var newXcon *crossconnect.CrossConnect
	// 9. We need to program dataplane with our values.
	// 9.1 Sending updated request to dataplane.
	for dpRetry := 0; dpRetry < DataplaneRetryCount; dpRetry++ {
		if ctx.Err() != nil {
			cce.doFailureClose(ctx)
			return nil, ctx.Err()
		}

		logger.Infof("NSM:(9.1) Sending request to dataplane: %v retry: %v", clientConnection.Xcon, dpRetry)
		dpCtx, cancel := context.WithTimeout(ctx, DataplaneTimeout)
		newXcon, err = dataplaneClient.Request(dpCtx, clientConnection.Xcon)
		cancel()
		if err != nil {
			logger.Errorf("NSM:(9.1.1) Dataplane request failed: %v retry: %v", err, dpRetry)

			// Let's try again with a short delay
			if dpRetry < DataplaneRetryCount-1 {
				<-time.After(DataplaneRetryDelay)
				continue
			}
			logger.Errorf("NSM:(9.1.2) Dataplane request  all retry attempts failed: %v", clientConnection.Xcon)
			// 9.3 If datplane configuration are failed, we need to close remore NSE actually.
			cce.doFailureClose(ctx)
			return conn, err
		}

		// In case of context deadline, we need to close NSE and dataplane.
		if ctx.Err() != nil {
			cce.doFailureClose(ctx)
			return nil, ctx.Err()
		}

		clientConnection.Xcon = newXcon

		logger.Infof("NSM:(9.2) Dataplane configuration successful %v", clientConnection.Xcon)
		break
	}

	// Update connection context if it updated from dataplane.
	return cce.updateClientConnection(ctx, conn, clientConnection, dp)
}

func (cce *dataplaneService) updateClientConnection(ctx context.Context, conn *connection.Connection, clientConnection *model.ClientConnection, dp *model.Dataplane) (*connection.Connection, error) {
	// Update connection context if it updated from dataplane.
	err := conn.UpdateContext(clientConnection.GetConnectionSource().GetContext())
	if err != nil {
		cce.doFailureClose(ctx)
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
