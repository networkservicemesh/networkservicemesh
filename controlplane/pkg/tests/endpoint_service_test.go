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
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/tests/mock"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	mechanismCommon "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

func TestRequestToLocalEndpointService(t *testing.T) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testEndpoint1 := &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name: "network_service",
		},
		NetworkServiceManager: &registry.NetworkServiceManager{
			Name: "nsm1",
		},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:                      "nse",
			NetworkServiceManagerName: "nsm1",
			NetworkServiceName:        "network_service",
		},
	}

	nseConn := &connection.Connection{
		Id:             "0",
		NetworkService: "network_service",
		Context:        &connectioncontext.ConnectionContext{IpContext: &connectioncontext.IPContext{}},
		Mechanism: &connection.Mechanism{
			Type:       vxlan.MECHANISM,
			Parameters: map[string]string{mechanismCommon.Workspace: "", kernel.WorkspaceNSEName: ""},
		},
	}

	// Mock NSE client
	nseClientMock := mock.NewMockNetworkServiceClient(ctrl)
	nseClientMock.EXPECT().Cleanup().Return(nil)
	nseClientMock.EXPECT().Request(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
		if request == nil {
			return nil, errors.Errorf("Request should not be nil")
		}
		return nseConn, nil
	})

	// Mock NSE manager
	nseManagerMock := mock.NewMockNetworkServiceEndpointManager(ctrl)
	nseManagerMock.EXPECT().IsLocalEndpoint(gomock.Any()).Return(true)
	nseManagerMock.EXPECT().CreateNSEClient(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceClient, error) {
			if endpoint != testEndpoint1 {
				return nil, errors.Errorf("Given endpoint doesn't equal to expected")
			}
			return nseClientMock, nil
		},
	)

	// Initialize context for the request
	ctx := common.WithForwarder(context.Background(), testForwarder1)
	ctx = common.WithLog(ctx, logrus.New())
	ctx = common.WithModelConnection(ctx, &model.ClientConnection{})
	ctx = common.WithEndpoint(ctx, testEndpoint1)

	// Initialize model
	testModel := newTestModel()
	testModel.AddEndpoint(context.Background(),
		&model.Endpoint{
			Endpoint:  testEndpoint1,
			Workspace: "workspace",
		},
	)

	builder := common.NewLocalRequestBuilder(testModel)
	service := common.NewEndpointService(nseManagerMock, nil, testModel, builder)

	request := &networkservice.NetworkServiceRequest{Connection: testConnection}
	conn, err := service.Request(ctx, request)

	g.Expect(err).To(BeNil())

	// Check that source connection context is updated and the same as dst context
	g.Expect(conn.Context).To(Equal(nseConn.Context))

	// Check nse connection parameters are updated
	g.Expect(nseConn.GetMechanism().GetParameters()[mechanismCommon.Workspace]).To(Equal("workspace"))
	g.Expect(nseConn.GetMechanism().GetParameters()[kernel.WorkspaceNSEName]).To(Equal("nse"))

	g.Expect(conn.Id).To(Equal(""))
}
