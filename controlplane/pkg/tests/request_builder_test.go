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
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

type testModel struct {
	model.Model
}

func (m *testModel) ConnectionID() string {
	return "1"
}

func newTestModel() *testModel {
	m := &testModel{
		Model: model.NewModel(),
	}
	testNsm := &registry.NetworkServiceManager{
		Name: "nsm1",
	}
	m.SetNsm(testNsm)
	return m
}

func createTestFwdWithLocalMechanism() *model.Forwarder {
	return &model.Forwarder{
		LocalMechanisms: []*connection.Mechanism{
			{
				Type: kernel.MECHANISM,
			},
		},
	}
}

func getTestFwdWithRemoteMechanism() *model.Forwarder {
	return &model.Forwarder{
		RemoteMechanisms: []*connection.Mechanism{
			{
				Type: vxlan.MECHANISM,
			},
		},
	}
}

func createLocalTestEndpoint() *registry.NSERegistration {
	return &registry.NSERegistration{
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
}

func createRemoteTestEndpoint() *registry.NSERegistration {
	return &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name: "network_service",
		},
		NetworkServiceManager: &registry.NetworkServiceManager{
			Name: "nsm2",
		},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:                      "nse",
			NetworkServiceManagerName: "nsm2",
			NetworkServiceName:        "network_service",
		},
	}
}

func getTestConnection() *connection.Connection {
	return &connection.Connection{
		Labels:         map[string]string{"key": "value"},
		NetworkService: "network_service",
	}
}

func TestLocalRequestBuilder_LocalNSERequest(t *testing.T) {
	g := NewWithT(t)

	builder := common.NewLocalRequestBuilder(newTestModel())

	request1 := builder.Build("0", createLocalTestEndpoint(), createTestFwdWithLocalMechanism(), getTestConnection())
	checkLocalNSERequest(g, request1)
	g.Expect(request1.Connection.Id).To(Equal("0"))

	// Check request if ID is absent.
	request2 := builder.Build("", createLocalTestEndpoint(), createTestFwdWithLocalMechanism(), getTestConnection())
	checkLocalNSERequest(g, request2)
	g.Expect(request2.Connection.Id).To(Equal("1"))
}

func TestLocalRequestBuilder_RemoteNSMRequest(t *testing.T) {
	g := NewWithT(t)

	builder := common.NewLocalRequestBuilder(newTestModel())

	request1 := builder.Build("0", createRemoteTestEndpoint(), getTestFwdWithRemoteMechanism(), getTestConnection())
	checkRemoteNSMRequest(g, request1)
	g.Expect(request1.Connection.Id).To(Equal("0"))

	// Check request if ID is absent.
	request2 := builder.Build("", createRemoteTestEndpoint(), getTestFwdWithRemoteMechanism(), getTestConnection())
	checkRemoteNSMRequest(g, request2)
	g.Expect(request2.Connection.Id).To(Equal(""))
}

func TestRemoteRequestBuilder(t *testing.T) {
	g := NewWithT(t)

	builder := common.NewRemoteRequestBuilder(newTestModel())

	request1 := builder.Build("0", createLocalTestEndpoint(), createTestFwdWithLocalMechanism(), getTestConnection())
	checkLocalNSERequest(g, request1)
	g.Expect(request1.Connection.Id).To(Equal("0"))

	// Check request if ID is absent.
	request2 := builder.Build("", createLocalTestEndpoint(), createTestFwdWithLocalMechanism(), getTestConnection())
	checkLocalNSERequest(g, request2)
	g.Expect(request2.Connection.Id).To(Equal("1"))
}

func checkLocalNSERequest(g *WithT, request *networkservice.NetworkServiceRequest) {
	g.Expect(request).NotTo(BeNil())
	g.Expect(request.Connection.NetworkService).To(Equal("network_service"))
	g.Expect(len(request.Connection.Labels)).To(Equal(1))
	g.Expect(request.Connection.Labels["key"]).NotTo(BeNil())
	g.Expect(request.Connection.Labels["key"]).To(Equal("value"))
	g.Expect(len(request.Connection.NetworkServiceManagers)).To(Equal(1))
	g.Expect(request.Connection.NetworkServiceManagers[0]).To(Equal("nsm1"))
	g.Expect(len(request.MechanismPreferences)).To(Equal(1))
	g.Expect(request.MechanismPreferences[0].Type).To(Equal(kernel.MECHANISM))
}

func checkRemoteNSMRequest(g *WithT, request *networkservice.NetworkServiceRequest) {
	g.Expect(request.Connection.NetworkService).To(Equal("network_service"))
	g.Expect(len(request.Connection.Labels)).To(Equal(1))
	g.Expect(request.Connection.Labels["key"]).NotTo(BeNil())
	g.Expect(request.Connection.Labels["key"]).To(Equal("value"))
	g.Expect(len(request.Connection.NetworkServiceManagers)).To(Equal(2))
	g.Expect(request.Connection.NetworkServiceManagers[0]).To(Equal("nsm1"))
	g.Expect(request.Connection.NetworkServiceManagers[1]).To(Equal("nsm2"))
	g.Expect(len(request.MechanismPreferences)).To(Equal(1))
	g.Expect(request.MechanismPreferences[0].Type).To(Equal(vxlan.MECHANISM))
}
