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
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
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

	id := request.GetRequestConnection().GetId()
	clientConnection := cce.model.GetClientConnection(id)

	if clientConnection != nil {
		// If one of in progress states, we need to exit with failure, since operation on this connection are in progress already.
		if clientConnection.ConnectionState == model.ClientConnectionRequesting ||
			clientConnection.ConnectionState == model.ClientConnectionHealing ||
			clientConnection.ConnectionState == model.ClientConnectionClosing {
			err := fmt.Errorf("trying to request connection in bad state")
			logger.Errorf("Error %v", err)
			return nil, err
		}

		request.Connection.SetID(clientConnection.GetID())
		logger.Infof("NSM:(%v) Called with existing connection passed: %v", id, clientConnection)

		clientConnection.ConnectionState = model.ClientConnectionHealing
		// Update model connection status
		cce.model.ApplyClientConnectionChanges(clientConnection.GetID(), func(modelCC *model.ClientConnection) {
			modelCC.ConnectionState = model.ClientConnectionHealing
		})
	} else {
		// Assign ID to connection
		request.Connection.SetID(cce.model.ConnectionID())

		clientConnection = &model.ClientConnection{
			ConnectionID:    request.Connection.GetId(),
			ConnectionState: model.ClientConnectionRequesting,
		}
		cce.model.AddClientConnection(clientConnection)
	}

	// 8. Remember original Request for Heal cases.
	clientConnection.Request = request
	ctx = common.WithModelConnection(ctx, clientConnection)

	conn, err := ProcessNext(ctx, request)
	if err != nil {
		// In case of error we need to remove it from model
		cce.model.DeleteClientConnection(clientConnection.GetID())
		return conn, err
	}

	clientConnection.Span = common.OriginalSpan(ctx)
	clientConnection.ConnectionState = model.ClientConnectionReady
	// 10. Send update for client connection
	cce.model.UpdateClientConnection(clientConnection)

	return conn, err
}

func (cce *connectionService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	logrus.Infof("Closing connection: %v", connection)

	clientConnection := cce.model.GetClientConnection(connection.GetId())
	if clientConnection == nil {
		err := fmt.Errorf("there is no such client connection %v", connection)
		logrus.Error(err)
		return nil, err
	}
	// Pass model connection with context
	ctx = common.WithModelConnection(ctx, clientConnection)

	_, err := ProcessClose(ctx, connection)

	cce.model.DeleteClientConnection(clientConnection.GetID())

	return &empty.Empty{}, err
}
