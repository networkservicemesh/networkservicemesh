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
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	tests "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/tests/mock"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
)

func createXCon() *crossconnect.CrossConnect {
	return &crossconnect.CrossConnect{
		Source: &connection.Connection{
			NetworkService: "nsm",
			Context:        &connectioncontext.ConnectionContext{IpContext: &connectioncontext.IPContext{}},
		},
		Destination: &connection.Connection{
			NetworkService: "nsm",
			Context: &connectioncontext.ConnectionContext{
				EthernetContext: &connectioncontext.EthernetContext{
					DstMac: "dst_mac", SrcMac: "src_mac",
				},
			},
		},
	}
}

func createForwarder() *model.Forwarder {
	return &model.Forwarder{
		RegisteredName: "fwd_registered_name",
		LocalMechanisms: []*connection.Mechanism{
			{
				Type: kernel.MECHANISM,
			},
		},
	}
}

func TestForwarderServiceRequest(t *testing.T) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)

	grpcClientConn, _ := grpc.Dial("")

	testXCon := createXCon()

	mdl := newModel()
	modelConnection := &model.ClientConnection{
		Xcon: testXCon,
	}

	forwarderClient := tests.NewMockForwarderClient(ctrl)
	forwarderClient.EXPECT().Request(gomock.Any(), gomock.Any()).Return(testXCon, nil)

	serviceReg := tests.NewMockServiceRegistry(ctrl)
	serviceReg.EXPECT().WaitForForwarderAvailable(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	serviceReg.EXPECT().ForwarderConnection(gomock.Any(), gomock.Any()).Return(
		forwarderClient, grpcClientConn, nil)

	ctx := context.Background()
	ctx = common.WithModelConnection(ctx, modelConnection)
	ctx = common.WithOriginalSpan(ctx, spanhelper.GetSpanHelper(ctx))
	mdl.AddForwarder(ctx, createForwarder())

	forwarderService := common.NewForwarderService(mdl, serviceReg, common.NewLocalMechanismSelector())

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "nsm",
		},
		MechanismPreferences: []*connection.Mechanism{
			{
				Type: kernel.MECHANISM,
			},
		},
	}

	conn, err := forwarderService.Request(ctx, request)

	g.Expect(err).To(BeNil())
	g.Expect(conn).NotTo(BeNil())

	// Check connection mechanisms and context are updated
	g.Expect(conn.Mechanism).NotTo(BeNil())
	g.Expect(conn.Mechanism.Type).To(Equal(kernel.MECHANISM))
	g.Expect(conn.Mechanism.Parameters).NotTo(BeNil())

	g.Expect(conn.Context).To(Equal(testXCon.Source.Context))

	// Check ethernet context
	g.Expect(conn.Context.EthernetContext).To(Equal(testXCon.Destination.Context.EthernetContext))

	//Check forwarder state == ForwarderStateReady
	g.Expect(modelConnection.ForwarderState).To(Equal(model.ForwarderStateReady))

	// Check forwarder registered name
	g.Expect(modelConnection.ForwarderRegisteredName).To(Equal("fwd_registered_name"))
}

func TestForwarderServiceClose(t *testing.T) {
	g := NewWithT(t)
	ctrl := gomock.NewController(t)

	grpcClientConn, _ := grpc.Dial("")

	mdl := newModel()
	modelConnection := &model.ClientConnection{
		ForwarderState:          model.ForwarderStateReady,
		ForwarderRegisteredName: "fwd_registered_name",
	}

	clientCloseCalled := false

	forwarderClient := tests.NewMockForwarderClient(ctrl)
	forwarderClient.EXPECT().Close(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *crossconnect.CrossConnect, _ ...grpc.CallOption) (*empty.Empty, error) {
		clientCloseCalled = true
		return &empty.Empty{}, nil
	})

	serviceReg := tests.NewMockServiceRegistry(ctrl)
	serviceReg.EXPECT().ForwarderConnection(gomock.Any(), gomock.Any()).Return(
		forwarderClient, grpcClientConn, nil)

	ctx := context.Background()
	ctx = common.WithModelConnection(ctx, modelConnection)
	ctx = common.WithOriginalSpan(ctx, spanhelper.GetSpanHelper(ctx))
	mdl.AddForwarder(ctx, createForwarder())

	forwarderService := common.NewForwarderService(mdl, serviceReg, common.NewLocalMechanismSelector())

	conn := &connection.Connection{
		NetworkService: "nsm",
	}

	_, err := forwarderService.Close(ctx, conn)

	g.Expect(err).To(BeNil())

	// Check close called
	g.Expect(clientCloseCalled).To(BeTrue())

	// Check forwarder state changed
	g.Expect(modelConnection.ForwarderState).To(Equal(model.ForwarderStateNone))
}
