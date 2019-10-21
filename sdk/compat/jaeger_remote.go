// Copyright (c) 2019 Cisco Systems, Inc.
//
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

package compat

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
)

type jaegerWrappedRemoteNetworkServiceServer struct {
	name string
	networkservice.NetworkServiceServer
}

func NewJaegerWrappedRemoteNetworkServiceServer(name string, srv networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	if srv == nil {
		return nil
	}
	return &jaegerWrappedRemoteNetworkServiceServer{name: name, NetworkServiceServer: srv}
}

func (l *jaegerWrappedRemoteNetworkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	span := spanhelper.FromContext(ctx, l.name+".Request")
	defer span.Finish()
	span.LogObject("request", request)
	response, err := l.NetworkServiceServer.Request(ctx, request)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", response)
	return response, err
}

func (l *jaegerWrappedRemoteNetworkServiceServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	span := spanhelper.FromContext(ctx, l.name+".Close")
	defer span.Finish()
	span.LogObject("request", conn)
	response, err := l.NetworkServiceServer.Close(ctx, conn)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", response)
	return response, err
}
