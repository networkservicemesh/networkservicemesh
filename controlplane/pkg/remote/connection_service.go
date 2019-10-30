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
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"

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
	logger := common.Log(ctx)
	logger.Infof("Received request from client to connect to NetworkService: %v", request)
	span := spanhelper.GetSpanHelper(ctx)
	id := request.GetRequestConnection().GetId()
	clientConnection := cce.model.GetClientConnection(id)

	if clientConnection != nil {
		// If one of in progress states, we need to exit with failure, since operation on this connection are in progress already.
		if clientConnection.ConnectionState == model.ClientConnectionRequesting ||
			clientConnection.ConnectionState == model.ClientConnectionHealing ||
			clientConnection.ConnectionState == model.ClientConnectionClosing {
			err := errors.Errorf("trying to request connection in bad state")
			span.LogError(err)
			return nil, err
		}

		request.Connection.Id = clientConnection.GetID()
		logger.Infof("NSM:(%v) Called with existing connection passed: %v", id, clientConnection)

		// Update model connection status
		clientConnection = cce.model.ApplyClientConnectionChanges(ctx, clientConnection.GetID(), func(modelCC *model.ClientConnection) {
			modelCC.ConnectionState = model.ClientConnectionHealing
			modelCC.Span = common.OriginalSpan(ctx)
		})
	} else {
		// Assign ID to connection
		request.Connection.Id = cce.model.ConnectionID()

		clientConnection = &model.ClientConnection{
			ConnectionID:    request.Connection.GetId(),
			ConnectionState: model.ClientConnectionRequesting,
			Span:            common.OriginalSpan(ctx),
			Monitor:         common.ConnectionMonitor(ctx),
		}
		cce.model.AddClientConnection(ctx, clientConnection)
	}

	// 8. Remember original Request for Heal cases.
	clientConnection.Request = request
	ctx = common.WithModelConnection(ctx, clientConnection)

	conn, err := ProcessNext(ctx, request)
	if err != nil {
		// In case of error we need to remove it from model
		cce.model.DeleteClientConnection(ctx, clientConnection.GetID())
		return conn, err
	}
	clientConnection.Span = common.OriginalSpan(ctx)
	clientConnection.ConnectionState = model.ClientConnectionReady
	// 10. Send update for client connection
	cce.model.UpdateClientConnection(ctx, clientConnection)

	return conn, err
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
	// Pass model connection with context
	ctx = common.WithModelConnection(ctx, clientConnection)

	_, err := ProcessClose(ctx, connection)

	if err != nil {
		logger.Error(err)
	}
	cce.model.DeleteClientConnection(ctx, clientConnection.GetID())

	// Return empty to send update
	return &empty.Empty{}, err
}
