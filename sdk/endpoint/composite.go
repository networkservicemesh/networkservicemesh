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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

// InitContext is the context passed to the Init function of the endpoint
type InitContext struct {
	GrpcServer *grpc.Server
}

// ChainedEndpoint is the basic endpoint composition interface
type ChainedEndpoint interface {
	networkservice.NetworkServiceServer
	Name() string
	Init(context *InitContext) error
	GetNext() ChainedEndpoint
	GetOpaque(interface{}) interface{}
	setNext(service ChainedEndpoint)
}

// BaseCompositeEndpoint is the base for building endpoints
type BaseCompositeEndpoint struct {
	next ChainedEndpoint
}

func (c *BaseCompositeEndpoint) setNext(service ChainedEndpoint) {
	c.next = service
}

// Init is called for each composite in the chain during NSM Endpoint instantiation
func (c *BaseCompositeEndpoint) Init(context *InitContext) error {
	return nil
}

// GetNext returns the next endpoint in the composition chain
func (c *BaseCompositeEndpoint) GetNext() ChainedEndpoint {
	return c.next
}

// GetOpaque is an implementation specific method to get arbitrary data out of a composite
func (c *BaseCompositeEndpoint) GetOpaque(interface{}) interface{} {
	return nil
}

// CompositeEndpoint is the base service composition struct
type CompositeEndpoint struct {
	chainedEndpoints []ChainedEndpoint
}

// Request implements a dummy request handler
func (bce *CompositeEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.NetworkServiceReply, error) {
	if len(bce.chainedEndpoints) == 0 {
		return &networkservice.NetworkServiceReply{Connection: request.Connection}, nil
	}
	return bce.chainedEndpoints[0].Request(ctx, request)
}

// Close implements a dummy close handler
func (bce *CompositeEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if len(bce.chainedEndpoints) == 0 {
		return &empty.Empty{}, nil
	}
	return bce.chainedEndpoints[0].Close(ctx, connection)
}

// NewCompositeEndpoint creates a new composed endpoint
func NewCompositeEndpoint(endpoints ...ChainedEndpoint) *CompositeEndpoint {
	for i := 0; i < len(endpoints); i++ {
		var nextEndpoint ChainedEndpoint
		if i != len(endpoints)-1 {
			nextEndpoint = endpoints[i+1]
		}
		endpoints[i].setNext(nextEndpoint)
	}
	return &CompositeEndpoint{
		chainedEndpoints: endpoints,
	}
}
