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

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
)

// CompositeEndpoint is  the base service compostion interface
type CompositeEndpoint interface {
	networkservice.NetworkServiceServer
	SetSelf(CompositeEndpoint)
	GetNext() CompositeEndpoint
	SetNext(CompositeEndpoint) CompositeEndpoint
	GetOpaque(interface{}) interface{}
}

// BaseCompositeEndpoint is the base service compostion struct
type BaseCompositeEndpoint struct {
	self CompositeEndpoint
	next CompositeEndpoint
}

// Request imeplements a dummy request handler
func (bce *BaseCompositeEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if bce.GetNext() != nil {
		return bce.GetNext().Request(ctx, request)
	}
	return nil, nil
}

// Close imeplements a dummy close handler
func (bce *BaseCompositeEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	if bce.GetNext() != nil {
		return bce.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// SetSelf sets the original self struct
func (bce *BaseCompositeEndpoint) SetSelf(self CompositeEndpoint) {
	bce.self = self
}

// GetNext returns the next composite
func (bce *BaseCompositeEndpoint) GetNext() CompositeEndpoint {
	return bce.next
}

// SetNext sets the next composite
func (bce *BaseCompositeEndpoint) SetNext(next CompositeEndpoint) CompositeEndpoint {
	if bce.self == nil {
		logrus.Fatal("Any struct that edtends BaseCompositeEndpoint should have 'self' set. Consider using SetSelf().")
	}
	bce.next = next
	return bce.self
}

// GetOpaque returns a composite specific arnitrary data
func (bce *BaseCompositeEndpoint) GetOpaque(interface{}) interface{} {
	return nil
}
