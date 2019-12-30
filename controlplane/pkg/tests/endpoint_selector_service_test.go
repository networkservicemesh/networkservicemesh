// Copyright (c) 2019 Cisco and/or its affiliates.
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

package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/local"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/tests/mock"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/remote"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
)

func createConnectionCtx() *connectioncontext.ConnectionContext {
	return &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			SrcIpAddr: "127.0.0.1",
			DstIpAddr: "127.0.0.2",
		},
	}
}

func createTestEndpoint(epName string) *registry.NSERegistration {
	return &registry.NSERegistration{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:                      epName,
			NetworkServiceName:        "ns",
			NetworkServiceManagerName: "nsmgr",
		},
		NetworkService: &registry.NetworkService{
			Name: "ns",
		},
		NetworkServiceManager: &registry.NetworkServiceManager{
			Name: "nsmgr",
			Url:  "nsmgr_url",
		},
	}
}

func createXCon() *crossconnect.CrossConnect {
	return &crossconnect.CrossConnect{
		Source: &connection.Connection{
			NetworkService: "ns",
			Context:        createConnectionCtx(),
		},
		Destination: &connection.Connection{
			Id:             "dst_conn_id",
			NetworkService: "ns",
			Context: &connectioncontext.ConnectionContext{
				EthernetContext: &connectioncontext.EthernetContext{
					DstMac: "dst_mac", SrcMac: "src_mac",
				},
			},
		},
	}
}

func TestRemoteEndpointSelectorService_Request(t *testing.T) {
	g := NewWithT(t)

	testEndpoint := createTestEndpoint("ep")

	modelConnection := &model.ClientConnection{}

	ctx := context.Background()
	ctx = common.WithModelConnection(ctx, modelConnection)
	ctx = common.WithOriginalSpan(ctx, spanhelper.GetSpanHelper(ctx))

	mdl := newModel()
	mdl.AddEndpoint(ctx, &model.Endpoint{Endpoint: testEndpoint})

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService:             "nsm",
			NetworkServiceEndpointName: testEndpoint.NetworkServiceEndpoint.Name,
		},
	}

	service := remote.NewEndpointSelectorService(mdl)
	conn, err := service.Request(ctx, request)

	g.Expect(err).To(BeNil())
	g.Expect(conn).NotTo(BeNil())

	// Check endpoint is put to client connection
	g.Expect(conn.NetworkServiceEndpointName).To(Equal(testEndpoint.NetworkServiceEndpoint.Name))

	// Check target endpoint is in model connection
	g.Expect(modelConnection.Endpoint).To(Equal(testEndpoint))
}

// Check request that is successful with first attempt
func TestLocalEndpointSelectorService_Request(t *testing.T) {
	testLocalSelectorServiceRequestWithAttempts(t, 1)
}

// Test local endpoint service request that is successful only with 3rd attempt
func TestLocalEndpointSelectorService_RequestWithIgnoredEndpoints(t *testing.T) {
	testLocalSelectorServiceRequestWithAttempts(t, 3)
}

func TestLocalEndpointSelectorService_RequestWithHealingConn(t *testing.T) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)

	testEndpoint := createTestEndpoint("ep")

	modelConnection := &model.ClientConnection{
		Endpoint:        testEndpoint,
		ConnectionState: model.ClientConnectionHealing,
		Xcon:            createXCon(),
	}
	ignoredEndpoints := make(map[registry.EndpointNSMName]*registry.NSERegistration)

	ctx := context.Background()
	ctx = common.WithModelConnection(ctx, modelConnection)
	ctx = common.WithOriginalSpan(ctx, spanhelper.GetSpanHelper(ctx))
	ctx = common.WithIgnoredEndpoints(ctx, ignoredEndpoints)

	nseManagerMock := mock.NewMockNetworkServiceEndpointManager(ctrl)
	nseManagerMock.EXPECT().IsLocalEndpoint(gomock.Any()).Return(true)
	nseManagerMock.EXPECT().GetEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(testEndpoint, nil)

	mdl := newModel()
	mdl.AddEndpoint(ctx, &model.Endpoint{Endpoint: testEndpoint})

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService:             "ns",
			NetworkServiceEndpointName: testEndpoint.NetworkServiceEndpoint.Name,
			Mechanism:                  &connection.Mechanism{Type: kernel.MECHANISM},
			Context:                    createConnectionCtx(),
		},
	}

	service := local.NewEndpointSelectorService(nseManagerMock)
	conn, err := service.Request(ctx, request)

	g.Expect(err).To(BeNil())
	g.Expect(conn).NotTo(BeNil())

	// Check request connection ctx is updated to Xcon dst context
	g.Expect(conn.GetContext()).To(Equal(modelConnection.GetConnectionDestination().GetContext()))

	// Check modelConnection updated
	g.Expect(modelConnection.GetConnectionSource().GetState()).To(Equal(connection.State_UP))
	g.Expect(modelConnection.GetConnectionSource().GetMechanism()).To(Equal(request.Connection.GetMechanism()))
}

func testLocalSelectorServiceRequestWithAttempts(t *testing.T, totalAttemptCount int) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)

	modelConnection := &model.ClientConnection{}
	ignoredEndpoints := make(map[registry.EndpointNSMName]*registry.NSERegistration)

	var endpoints []*registry.NSERegistration

	nseManagerMock := mock.NewMockNetworkServiceEndpointManager(ctrl)
	nseManagerMock.EXPECT().IsLocalEndpoint(gomock.Any()).Return(true)

	// Imitate selecting endpoint for particular network service few times
	for i := 0; i < totalAttemptCount; i++ {
		ep := createTestEndpoint(fmt.Sprintf("ep%d", i))
		nseManagerMock.EXPECT().GetEndpoint(gomock.Any(), gomock.Any(), gomock.Any()).Return(ep, nil)
		endpoints = append(endpoints, ep)
	}

	// Imitate fail request to NSE few times, and then success
	nextServiceMock := mock.NewMockNetworkServiceServer(ctrl)
	nextServiceMock.EXPECT().Request(gomock.Any(), gomock.Any()).Return(nil, errors.Errorf("Request failed")).Times(totalAttemptCount - 1)
	nextServiceMock.EXPECT().Request(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, _ *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
		// Check ctx contains target endpoint
		g.Expect(common.Endpoint(ctx)).To(Equal(endpoints[totalAttemptCount-1]))
		return &connection.Connection{
			NetworkService:             "ns",
			NetworkServiceEndpointName: common.Endpoint(ctx).NetworkServiceEndpoint.Name,
			Mechanism:                  &connection.Mechanism{Type: kernel.MECHANISM},
			Context:                    createConnectionCtx(),
		}, nil
	})

	ctx := context.Background()
	ctx = common.WithModelConnection(ctx, modelConnection)
	ctx = common.WithOriginalSpan(ctx, spanhelper.GetSpanHelper(ctx))
	ctx = common.WithIgnoredEndpoints(ctx, ignoredEndpoints)
	ctx = common.WithNext(ctx, nextServiceMock)

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "ns",
		},
	}

	service := local.NewEndpointSelectorService(nseManagerMock)
	conn, err := service.Request(ctx, request)

	g.Expect(err).To(BeNil())
	g.Expect(conn).NotTo(BeNil())

	// Check target endpoint is in model connection
	g.Expect(modelConnection.Endpoint).To(Equal(endpoints[totalAttemptCount-1]))

	// Check ignoredEndpoints contains ep that were failed
	g.Expect(len(ignoredEndpoints)).To(Equal(totalAttemptCount - 1))
	for i := 0; i < totalAttemptCount-1; i++ {
		g.Expect(ignoredEndpoints[endpoints[i].GetEndpointNSMName()]).To(Equal(endpoints[i]))
	}
}
