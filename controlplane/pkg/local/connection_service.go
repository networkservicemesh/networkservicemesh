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
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
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
	span := opentracing.SpanFromContext(ctx)

	id := request.GetRequestConnection().GetId()
	if span != nil {
		span.LogFields(log.String("connection-id", id))
	}

	workspaceName := common.WorkspaceName(ctx)

	if span != nil {
		span.LogFields(log.String("workspace", workspaceName))
	}

	// We need to take updated connection in case of updates
	clientConnection := common.ModelConnection(ctx) // Case only for Healing.

	if clientConnection == nil {
		clientConnection = cce.model.GetClientConnection(id)
	} else if cce.model.GetClientConnection(id) == nil {
		err := fmt.Errorf("trying to request not existing connection")
		logger.Errorf("Error %v", err)
		return nil, err
	}
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
	clientConnection.Workspace = workspaceName

	if span != nil {
		span.LogFields(log.Object("client-connection", clientConnection))
	}
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
	cce.model.ApplyClientConnectionChanges(clientConnection.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionClosing
	})

	// Pass model connection with context
	ctx = common.WithModelConnection(ctx, clientConnection)

	_, err := ProcessClose(ctx, connection)

	cce.model.DeleteClientConnection(clientConnection.GetID())

	return &empty.Empty{}, err
}
