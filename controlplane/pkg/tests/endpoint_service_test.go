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

// TODO: replace all these with mocks
type testNseManager struct {
	nsm.NetworkServiceManager
	nseConnection    *connection.Connection
	expectedEndpoint *registry.NSERegistration
}

type testNSEClient struct {
	nsm.NetworkServiceClient
	nseConnection *connection.Connection
}

func (nseClient *testNSEClient) Request(_ context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if request == nil {
		return nil, errors.Errorf("Request should not be nil")
	}
	return nseClient.nseConnection, nil
}

func (nseClient *testNSEClient) Cleanup() error {
	return nil
}

func (nseM *testNseManager) CreateNSEClient(_ context.Context, endpoint *registry.NSERegistration) (nsm.NetworkServiceClient, error) {
	if endpoint != nseM.expectedEndpoint {
		return nil, errors.Errorf("Given endpoint doesn't equal to expected")
	}
	return &testNSEClient{
		nseConnection: nseM.nseConnection,
	}, nil
}

func (nseM *testNseManager) IsLocalEndpoint(_ *registry.NSERegistration) bool {
	return true
}

func (nseM *testNseManager) GetEndpoint(_ context.Context, _ *connection.Connection, _ map[registry.EndpointNSMName]*registry.NSERegistration) (*registry.NSERegistration, error) {
	return nil, nil
}

func (nseM *testNseManager) CheckUpdateNSE(_ context.Context, _ *registry.NSERegistration) bool {
	return false
}

func TestRequestToLocalEndpointService(t *testing.T) {
	g := NewWithT(t)

	// Initialize context for the request
	ctx := common.WithForwarder(context.Background(), testForwarder1)
	ctx = common.WithLog(ctx, logrus.New())
	ctx = common.WithModelConnection(ctx, &model.ClientConnection{})

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

	ctx = common.WithEndpoint(ctx, testEndpoint1)

	builder := common.NewLocalRequestBuilder(newTestModel())

	nseConn := &connection.Connection{
		Id:             "0",
		NetworkService: "network_service",
		Context:        &connectioncontext.ConnectionContext{IpContext: &connectioncontext.IPContext{}},
		Mechanism: &connection.Mechanism{
			Type:       vxlan.MECHANISM,
			Parameters: map[string]string{mechanismCommon.Workspace: "", kernel.WorkspaceNSEName: ""},
		},
	}

	nseManager := &testNseManager{nseConnection: nseConn, expectedEndpoint: testEndpoint1}

	testModel := newTestModel()
	testModel.AddEndpoint(context.Background(),
		&model.Endpoint{
			Endpoint:  testEndpoint1,
			Workspace: "workspace",
		},
	)

	service := common.NewEndpointService(nseManager, nil, testModel, builder)
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
