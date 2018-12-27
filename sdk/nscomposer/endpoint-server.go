// Copyright 2018 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
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

package nscomposer

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/sirupsen/logrus"
)

func (nsme *nsmEndpoint) outgoingConnectionRequest(ctx context.Context, request *networkservice.NetworkServiceRequest) (*nsmClient, error) {
	client, err := NewNSMClient(ctx, nsme.configuration, &dummyClientBackend{})
	if err != nil {
		logrus.Errorf("Unable to create the NSM client %v", err)
		return nil, err
	}

	client.name = client.name + request.GetConnection().GetId()
	client.mechanismType = request.GetMechanismPreferences()[0].GetType()
	client.Connect()

	// TODO: check this. Hack??
	client.GetConnection().GetMechanism().GetParameters()[connection.Workspace] = ""

	return client, nil
}

func (nsme *nsmEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Request for Network Service received %v", request)

	var client *nsmClient
	var err error
	if len(nsme.configuration.OutgoingNscName) > 0 {
		client, err = nsme.outgoingConnectionRequest(ctx, request)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
	}
	outgoingConnection := client.GetConnection()
	logrus.Infof("outgoingConnection: %v", outgoingConnection)

	incomingConnection, err := nsme.CompleteConnection(request, outgoingConnection)
	logrus.Infof("Completed incomingConnection %v", incomingConnection)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	err = nsme.backend.Request(ctx, incomingConnection, outgoingConnection, nsme.configuration.workspace)
	if err != nil {
		logrus.Errorf("The backend returned and error: %v", err)
		return nil, err
	}

	nsme.ioConnMap[incomingConnection] = client
	nsme.monitorConnectionServer.UpdateConnection(incomingConnection)
	logrus.Infof("Responding to NetworkService.Request(%v): %v", request, incomingConnection)
	return incomingConnection, nil
}

func (nsme *nsmEndpoint) Close(ctx context.Context, incomingConnection *connection.Connection) (*empty.Empty, error) {
	if outgoingConnection, ok := nsme.ioConnMap[incomingConnection]; ok {
		nsme.nsClient.Close(ctx, outgoingConnection.GetConnection())
	}
	nsme.backend.Close(ctx, incomingConnection, nsme.configuration.workspace)
	nsme.nsClient.Close(ctx, incomingConnection)
	nsme.monitorConnectionServer.DeleteConnection(incomingConnection)
	return &empty.Empty{}, nil
}
