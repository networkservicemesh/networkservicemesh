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

type NetworkService interface {
	networkservice.NetworkServiceServer
	GetOpaque(interface{}) interface{}
}

type Chained interface {
	SetNext(service NetworkService)
	GetNext() NetworkService
}

type ChainedService interface {
	NetworkService
	Chained
}

type ChainedImpl struct {
	next NetworkService
}

func (c *ChainedImpl) SetNext(service NetworkService) {
	c.next = service
}

func (c *ChainedImpl) GetNext() NetworkService {
	return c.next
}

// BaseCompositeEndpoint is the base service compostion struct
type CompositeService struct {
	serviceChain []ChainedService
}

func NewCompositeService(services ...ChainedService) *CompositeService {
	for i := 0; i < len(services); i++ {
		var nextService ChainedService
		if i != len(services)-1 {
			nextService = services[i+1]
		}
		services[i].SetNext(nextService)
	}
	return &CompositeService{
		serviceChain: services,
	}
}

// Request imeplements a dummy request handler
func (bce *CompositeService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if len(bce.serviceChain) == 0 {
		return request.Connection, nil
	}
	return bce.serviceChain[0].Request(ctx, request)
}

// Close imeplements a dummy close handler
func (bce *CompositeService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if len(bce.serviceChain) == 0 {
		return &empty.Empty{}, nil
	}
	return bce.serviceChain[0].Close(ctx, connection)
}

// GetOpaque returns a composite specific arnitrary data
func (bce *CompositeService) GetOpaque(interface{}) interface{} {
	return nil
}
