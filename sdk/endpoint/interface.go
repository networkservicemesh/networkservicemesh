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
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

// CompositeEndpoint is  the basic service compostion interface
type CompositeEndpoint interface {
	networkservice.NetworkServiceServer
	GetNext() CompositeEndpoint
	SetNext(CompositeEndpoint)
	GetOpaque(interface{}) interface{}
}

// BaseCompositeEndpoint is the base service compostion struct
type BaseCompositeEndpoint struct {
	next CompositeEndpoint
}

func (dce *BaseCompositeEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if dce.GetNext() != nil {
		return dce.GetNext().Request(ctx, request)
	}
	return nil, nil
}

func (dce *BaseCompositeEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if dce.GetNext() != nil {
		return dce.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// SetNext sets the next composite
func (bce *BaseCompositeEndpoint) SetNext(next CompositeEndpoint) {
	bce.next = next
}

func (bce *BaseCompositeEndpoint) GetNext() CompositeEndpoint {
	return bce.next
}

func (bce *BaseCompositeEndpoint) GetOpaque(interface{}) interface{} {
	return nil
}
