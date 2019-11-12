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

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

type connectionService struct {
	model model.Model
}

// NewConnectionService - creates a service to create and update model connection.
func NewConnectionService(model model.Model) networkservice.NetworkServiceServer {
	return &connectionService{
		model: model,
	}
}

func (cce *connectionService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	span := spanhelper.GetSpanHelper(ctx)
	span.Logger().Infof("Received request from client to connect to NetworkService: %v", request)

	id := request.GetRequestConnection().GetId()
	span.LogValue("connection-id", id)

	workspaceName := common.WorkspaceName(ctx)
	span.LogValue("workspace", workspaceName)

	// We need to take updated connection in case of updates
	clientConnection := common.ModelConnection(ctx) // Case only for Healing.

	// If healing, we should not remove connection from model
	healing := clientConnection != nil
	if clientConnection == nil {
		clientConnection = cce.model.GetClientConnection(id)
	} else if cce.model.GetClientConnection(id) == nil {
		err := errors.Errorf("trying to request not existing connection")
		span.LogError(err)
		return nil, err
	}
	if clientConnection != nil {
		// If one of in progress states, we need to exit with failure, since operation on this connection are in progress already.
		if cce.isConnectionInProgress(clientConnection) {
			err := errors.Errorf("trying to request connection in bad state")
			span.LogError(err)
			return nil, err
		}

		request.Connection.Id = clientConnection.GetID()
		clientConnection = cce.updateClientConnection(ctx, span.Logger(), id, clientConnection)
	} else {
		// Assign ID to connection
		request.Connection.Id = cce.model.ConnectionID()

		clientConnection = cce.createClientConnection(ctx, request)
		cce.model.AddClientConnection(ctx, clientConnection)
	}

	// 8. Remember original Request for Heal cases.
	clientConnection.Request = request
	ctx = common.WithModelConnection(ctx, clientConnection)

	conn, err := common.ProcessNext(ctx, request)
	if err != nil {
		if !healing {
			// In case of error we need to remove it from model
			cce.model.DeleteClientConnection(ctx, clientConnection.GetID())
		}
		span.Logger().Infof("Do not close connection since healing")
		return conn, err
	}

	clientConnection.ConnectionState = model.ClientConnectionReady
	span.LogObject("client-connection", clientConnection)

	// 10. Send update for client connection
	cce.model.UpdateClientConnection(ctx, clientConnection)

	return conn, err
}

func (cce *connectionService) isConnectionInProgress(clientConnection *model.ClientConnection) bool {
	return clientConnection.ConnectionState == model.ClientConnectionRequesting ||
		clientConnection.ConnectionState == model.ClientConnectionHealing ||
		clientConnection.ConnectionState == model.ClientConnectionClosing
}

func (cce *connectionService) updateClientConnection(ctx context.Context, logger logrus.FieldLogger, id string, clientConnection *model.ClientConnection) *model.ClientConnection {
	logger.Infof("NSM:(%v) Called with existing connection passed: %v", id, clientConnection)

	// Update model connection status
	clientConnection = cce.model.ApplyClientConnectionChanges(ctx, clientConnection.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionHealing
		modelCC.Span = common.OriginalSpan(ctx)
	})
	return clientConnection
}

func (cce *connectionService) createClientConnection(ctx context.Context, request *networkservice.NetworkServiceRequest) *model.ClientConnection {
	return &model.ClientConnection{
		ConnectionID:    request.Connection.GetId(),
		ConnectionState: model.ClientConnectionRequesting,
		Span:            common.OriginalSpan(ctx),
		Monitor:         common.ConnectionMonitor(ctx),
	}
}

func (cce *connectionService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	logger := common.Log(ctx)
	logger.Infof("Closing connection: %v", connection)

	clientConnection := cce.model.GetClientConnection(connection.GetId())
	if clientConnection == nil {
		err := errors.Errorf("there is no such client connection %v", connection)
		logger.Error(err)
		return nil, err
	}
	if clientConnection.ConnectionState == model.ClientConnectionClosing {
		return nil, errors.New("connection could not be closed, it is already closing")
	}

	clientConnection = cce.model.ApplyClientConnectionChanges(ctx, clientConnection.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionClosing
	})

	// Pass model connection with context
	ctx = common.WithModelConnection(ctx, clientConnection)

	_, err := common.ProcessClose(ctx, connection)

	if err != nil {
		logger.Error(err)
	}
	cce.model.DeleteClientConnection(ctx, clientConnection.GetID())

	// Return empty to send update
	return &empty.Empty{}, err
}
