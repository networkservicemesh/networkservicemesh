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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
)

type jaegerWrappedNetworkServiceServer struct {
	name string
	networkservice.NetworkServiceServer
}

func NewJaegerWrappedNetworkServiceServer(name string, srv networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return &jaegerWrappedNetworkServiceServer{name: name, NetworkServiceServer: srv}
}

func (j *jaegerWrappedNetworkServiceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	span := spanhelper.FromContext(ctx, j.name+".Request")
	defer span.Finish()
	span.LogObject("request", request)
	response, err := j.NetworkServiceServer.Request(ctx, request)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", response)
	return response, err
}

func (j *jaegerWrappedNetworkServiceServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	span := spanhelper.FromContext(ctx, j.name+".Close")
	defer span.Finish()
	span.LogObject("request", conn)
	response, err := j.NetworkServiceServer.Close(ctx, conn)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", response)
	return response, err
}

type jaegerWrappedLocalNetworkServiceClient struct {
	name string
	local.NetworkServiceClient
}

func NewJaegerWrappedLocalNetworkServiceClient(name string, client local.NetworkServiceClient) local.NetworkServiceClient {
	return &jaegerWrappedLocalNetworkServiceClient{name: name, NetworkServiceClient: client}
}

func (l *jaegerWrappedLocalNetworkServiceClient) Request(ctx context.Context, request *local.NetworkServiceRequest, opts ...grpc.CallOption) (*local_connection.Connection, error) {
	span := spanhelper.FromContext(ctx, l.name+".Request")
	defer span.Finish()
	span.LogObject("request", request)
	response, err := l.NetworkServiceClient.Request(ctx, request)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", response)
	return response, nil
}

func (l *jaegerWrappedLocalNetworkServiceClient) Close(ctx context.Context, conn *local_connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	span := spanhelper.FromContext(ctx, l.name+".Close")
	defer span.Finish()
	span.LogObject("request", conn)
	response, err := l.NetworkServiceClient.Close(ctx, conn)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", response)
	return response, err
}

type jaegerWrappedRemoteNetworkServiceClient struct {
	name string
	remote.NetworkServiceClient
}

func NewJaegerWrappedRemoteNetworkServiceClient(name string, client remote.NetworkServiceClient) remote.NetworkServiceClient {
	return &jaegerWrappedRemoteNetworkServiceClient{name: name, NetworkServiceClient: client}
}

func (l *jaegerWrappedRemoteNetworkServiceClient) Request(ctx context.Context, request *remote.NetworkServiceRequest, opts ...grpc.CallOption) (*remote_connection.Connection, error) {
	span := spanhelper.FromContext(ctx, l.name+".Request")
	defer span.Finish()
	span.LogObject("request", request)
	response, err := l.NetworkServiceClient.Request(ctx, request)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", response)
	return response, nil
}

func (l *jaegerWrappedRemoteNetworkServiceClient) Close(ctx context.Context, conn *remote_connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	span := spanhelper.FromContext(ctx, l.name+".Close")
	defer span.Finish()
	span.LogObject("request", conn)
	response, err := l.NetworkServiceClient.Close(ctx, conn)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", response)
	return response, err
}
