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
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

type Chained interface {
	SetNext(service ChainedEndpoint)
	GetNext() ChainedEndpoint
	GetOpaque(interface{}) interface{}
}

type ChainedEndpoint interface {
	networkservice.NetworkServiceServer
	Chained
}

type ChainedImpl struct {
	next ChainedEndpoint
}

func (c *ChainedImpl) SetNext(service ChainedEndpoint) {
	c.next = service
}

func (c *ChainedImpl) GetNext() ChainedEndpoint {
	return c.next
}

func (c *ChainedImpl) GetOpaque(interface{}) interface{} {
	return nil
}

// CompositeEndpoint is the base service composition struct
type CompositeEndpoint struct {
	chainedEndpoints []ChainedEndpoint
}

func NewCompositeEndpoint(endpoints ...ChainedEndpoint) *CompositeEndpoint {
	for i := 0; i < len(endpoints); i++ {
		var nextEndpoint ChainedEndpoint
		if i != len(endpoints)-1 {
			nextEndpoint = endpoints[i+1]
		}
		endpoints[i].SetNext(nextEndpoint)
	}
	return &CompositeEndpoint{
		chainedEndpoints: endpoints,
	}
}

// Request implements a dummy request handler
func (bce *CompositeEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if len(bce.chainedEndpoints) == 0 {
		return request.Connection, nil
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
