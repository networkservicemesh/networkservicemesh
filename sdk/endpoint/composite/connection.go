// Copyright 2018, 2019 VMware, Inc.
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

package composite

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/sdk/common"
	"github.com/ligato/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
	"github.com/teris-io/shortid"
)

type ConnectionCompositeEndpoint struct {
	endpoint.BaseCompositeEndpoint
	mechanismType connection.MechanismType
	id            *shortid.Shortid
}

func (cce *ConnectionCompositeEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	err := request.IsValid()
	if err != nil {
		logrus.Errorf("Request is not valid: %v", err)
		return nil, err
	}

	mechanism, err := connection.NewMechanism(cce.mechanismType, "nsm"+cce.id.MustGenerate(), "NSM Endpoint")
	if err != nil {
		logrus.Errorf("Mechanism not created: %v", err)
		return nil, err
	}

	var newConnection *connection.Connection
	if cce.GetNext() != nil {
		newConnection, err = cce.GetNext().Request(ctx, request)
		if err != nil {
			logrus.Errorf("Next request failed: %v", err)
			return nil, err
		}
	} else {
		newConnection = &connection.Connection{
			Id:             request.GetConnection().GetId(),
			NetworkService: request.GetConnection().GetNetworkService(),
			Mechanism:      mechanism,
			Context:        proto.Clone(request.Connection.Context).(*connectioncontext.ConnectionContext),
		}
	}

	if newConnection == nil {
		err := fmt.Errorf("Unabel to create a new connection")
		logrus.Errorf("%v", err)
		return nil, err
	}

	logrus.Infof("New connection created: %v", newConnection)
	return newConnection, nil
}

func (cce *ConnectionCompositeEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if cce.GetNext() != nil {
		return cce.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

func NewConnectionCompositeEndpoint(configuration *common.NSConfiguration) *ConnectionCompositeEndpoint {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	rand.Seed(time.Now().UTC().UnixNano())

	self := &ConnectionCompositeEndpoint{
		mechanismType: common.MechanismFromString(configuration.MechanismType),
		id:            shortid.MustNew(1, shortid.DEFAULT_ABC, rand.Uint64()),
	}
	self.SetSelf(self)

	return self
}
