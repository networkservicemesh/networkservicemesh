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

package trace

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
	"github.com/networkservicemesh/networkservicemesh/utils/typeutils"
)

type traceDiscoveryClient struct {
	traced registry.NetworkServiceDiscoveryClient
}

func NewNetworkServiceDiscoveryClient(traced registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient {
	return &traceDiscoveryClient{traced: traced}
}

func (t *traceDiscoveryClient) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest, opts ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	// Create a new span
	span := spanhelper.FromContext(ctx, fmt.Sprintf("%s.Request", typeutils.GetTypeName(t.traced)))
	defer span.Finish()

	// Make sure we log to span

	ctx = withLog(span.Context(), span.Logger())

	span.LogObject("request", request)

	// Actually call the next
	rv, err := t.traced.FindNetworkService(ctx, request, opts...)
	if err != nil {
		span.LogError(err)
		return nil, err
	}
	span.LogObject("response", rv)
	return rv, err
}
