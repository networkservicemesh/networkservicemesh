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
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
)

// CompositeNetworkService is the base service composition struct
type CompositeNetworkService struct {
	services []networkservice.NetworkServiceServer
}

// Request implements a dummy request handler
func (cns *CompositeNetworkService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if len(cns.services) == 0 {
		return request.Connection, nil
	}
	ctx = WithNext(ctx, &nextEndpoint{composite: cns, index: 0})
	if opentracing.IsGlobalTracerRegistered() {
		ctx = common.WithOriginalSpan(ctx, opentracing.SpanFromContext(ctx))
	}
	return cns.services[0].Request(ctx, request)
}

// Close implements a dummy close handler
func (cns *CompositeNetworkService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if len(cns.services) == 0 {
		return &empty.Empty{}, nil
	}
	ctx = WithNext(ctx, &nextEndpoint{composite: cns, index: 0})
	return cns.services[0].Close(ctx, connection)
}

// NewCompositeService creates a new composed endpoint
func NewCompositeService(services ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return &CompositeNetworkService{
		services: services,
	}
}
