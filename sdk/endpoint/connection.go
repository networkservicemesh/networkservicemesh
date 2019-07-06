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

package endpoint

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"github.com/teris-io/shortid"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

// ConnectionEndpoint makes basic Mechanism selection for the incoming connection
type ConnectionEndpoint struct {
	mechanismType connection.MechanismType
	// TODO - id doesn't seem to be used, and should be
	id *shortid.Shortid
}

// Request implements the request handler
// Consumes from ctx context.Context:
//	   Next
func (cce *ConnectionEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	err := request.IsValid()
	if err != nil {
		logrus.Errorf("Request is not valid: %v", err)
		return nil, err
	}

	mechanism, err := connection.NewMechanism(cce.mechanismType, cce.generateIfName(), "NSM Endpoint")
	if err != nil {
		logrus.Errorf("Mechanism not created: %v", err)
		return nil, err
	}

	request.GetConnection().Mechanism = mechanism

	if Next(ctx) != nil {
		return Next(ctx).Request(ctx, request)
	}
	return request.GetConnection(), nil
}

// Close implements the close handler
// Consumes from ctx context.Context:
//	   Next
func (cce *ConnectionEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// Name returns the composite name
func (cce *ConnectionEndpoint) Name() string {
	return "connection"
}

func (cce *ConnectionEndpoint) generateIfName() string {
	ifName := "nsm" + cce.id.MustGenerate()
	ifName = strings.Replace(ifName, "-", "", -1)
	ifName = strings.Replace(ifName, "_", "", -1)

	return ifName
}

// NewConnectionEndpoint creates a ConnectionEndpoint
func NewConnectionEndpoint(configuration *common.NSConfiguration) *ConnectionEndpoint {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	rand.Seed(time.Now().UTC().UnixNano())

	self := &ConnectionEndpoint{
		mechanismType: common.MechanismFromString(configuration.MechanismType),
		id:            shortid.MustNew(1, shortid.DefaultABC, rand.Uint64()),
	}

	return self
}
