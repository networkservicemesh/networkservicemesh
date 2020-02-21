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

// Package common - define a common set of services for both local/remote chains
package common

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/utils/typeutils"
)

const nextKey ContextKeyType = "Next"

type nextEndpoint struct {
	composite   *CompositeNetworkService
	index       int
	factoryName string
}

// WithNext -
//    Wraps 'parent' in a new Context that has the Next networkservice.NetworkServiceServer to be called in the chain
//    Should only be set in CompositeNetworkService.Request/Close
func WithNext(parent context.Context, next networkservice.NetworkServiceServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, nextKey, next)
}

// Next -
//   Returns the Next networkservice.NetworkServiceServer to be called in the chain from the context.Context
func Next(ctx context.Context) networkservice.NetworkServiceServer {
	if rv, ok := ctx.Value(nextKey).(networkservice.NetworkServiceServer); ok {
		return rv
	}
	return nil
}

// ProcessNext - performs a next operation on chain if defined.
func ProcessNext(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if Next(ctx) != nil {
		return Next(ctx).Request(ctx, request)
	}
	return request.Connection, nil
}

// ProcessClose - perform a next close operation on chain if defined
func ProcessClose(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

func (n *nextEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if n.index+1 < len(n.composite.services) {
		ctx = WithNext(ctx, &nextEndpoint{factoryName: n.factoryName, composite: n.composite, index: n.index + 1})
	} else {
		ctx = WithNext(ctx, nil)
	}

	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.%s.Request", n.factoryName, typeutils.GetTypeName(n.composite.services[n.index])))
	defer span.Finish()
	logger := span.Logger()
	ctx = span.Context()

	ctx = WithLog(ctx, logger)
	span.LogObject("request", request)

	// Actually call the next
	rv, err := n.composite.services[n.index].Request(ctx, request)

	span.LogError(err)
	span.LogObject("response", rv)
	return rv, err
}

func (n *nextEndpoint) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	if n.index+1 < len(n.composite.services) {
		ctx = WithNext(ctx, &nextEndpoint{factoryName: n.factoryName, composite: n.composite, index: n.index + 1})
	} else {
		ctx = WithNext(ctx, nil)
	}
	// Create a new span
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.%s.Close", n.factoryName, typeutils.GetTypeName(n.composite.services[n.index])))
	defer span.Finish()
	ctx = span.Context()

	// Make sure we log to span
	logger := span.Logger()
	ctx = WithLog(ctx, logger)

	span.LogObject("request", connection)
	rv, err := n.composite.services[n.index].Close(ctx, connection)

	span.LogError(err)
	span.LogObject("response", rv)
	return rv, err
}
