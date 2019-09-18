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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/utils/typeutils"
)

// InitContext is the context passed to the Init function of the endpoint
type InitContext struct {
	GrpcServer *grpc.Server
}

// Initable - things can be initted
type Initable interface {
	Init(*InitContext) error
}

// Init - initialize the thing if its initable, otherwise just fall through silently
func Init(thing interface{}, initContext *InitContext) error {
	if initialize, ok := thing.(Initable); ok {
		if err := initialize.Init(initContext); err != nil {
			logrus.Errorf("Failed to Init: %s -> err: %v", typeutils.GetTypeName(thing), err)
			return err
		}
	}
	return nil
}

// CompositeEndpoint is the base service composition struct
type CompositeEndpoint struct {
	endpoints []networkservice.NetworkServiceServer
}

// Request implements a dummy request handler
func (bce *CompositeEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if len(bce.endpoints) == 0 {
		return request.Connection, nil
	}
	ctx = withNext(ctx, &nextEndpoint{composite: bce, index: 0})
	return bce.endpoints[0].Request(ctx, request)
}

// Close implements a dummy close handler
func (bce *CompositeEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if len(bce.endpoints) == 0 {
		return &empty.Empty{}, nil
	}
	ctx = withNext(ctx, &nextEndpoint{composite: bce, index: 0})
	return bce.endpoints[0].Close(ctx, connection)
}

// Init initializes all the subcomponents of the composite
func (bce *CompositeEndpoint) Init(initContext *InitContext) error {
	for _, endpoint := range bce.endpoints {
		if err := Init(endpoint, initContext); err != nil {
			return err
		}
	}
	return nil
}

// NewCompositeEndpoint creates a new composed endpoint
func NewCompositeEndpoint(endpoints ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return &CompositeEndpoint{
		endpoints: endpoints,
	}
}
