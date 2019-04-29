// Copyright 2018 VMware, Inc.
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

package main

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
	"github.com/sirupsen/logrus"
)

const (
	defaultVPPAgentEndpoint = "localhost:9112"
)

type crossConnectStruct struct {
	crossConnect  *crossconnect.CrossConnect
	ingressIfName string
}

type vppAgentXConnComposite struct {
	endpoint.BaseCompositeEndpoint
	vppAgentEndpoint string
	crossConnects    map[string]crossConnectStruct
	workspace        string
}

func (vxc *vppAgentXConnComposite) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if vxc.GetNext() == nil {
		logrus.Fatal("Should have Next set")
	}

	incoming, err := vxc.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	opaque := vxc.GetNext().GetOpaque(incoming)
	if opaque == nil {
		err := fmt.Errorf("Backend: Unable to find the outgoing connection")
		logrus.Errorf("%v", err)
		return nil, err
	}
	outgoing := opaque.(*connection.Connection)

	incoming.Context = outgoing.GetContext()

	crossConnectRequest := &crossconnect.CrossConnect{
		Id:      incoming.GetId(),
		Payload: "IP",
		Source: &crossconnect.CrossConnect_LocalSource{
			LocalSource: incoming,
		},
		Destination: &crossconnect.CrossConnect_LocalDestination{
			LocalDestination: outgoing,
		},
	}

	crossConnect, dataChange, err := vxc.crossConnecVppInterfaces(ctx, crossConnectRequest, true, vxc.workspace)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	// The Crossconnect converter generates and puts the Source Interface name here
	ingressIfName := dataChange.VppConfig.XconnectPairs[0].ReceiveInterface

	// Store for cleanup
	vxc.crossConnects[incoming.GetId()] = crossConnectStruct{
		crossConnect:  crossConnect,
		ingressIfName: ingressIfName,
	}
	return incoming, nil
}

func (vxc *vppAgentXConnComposite) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	// remove from connections
	crossConnect, ok := vxc.crossConnects[conn.GetId()]
	if ok {
		_, _, err := vxc.crossConnecVppInterfaces(ctx, crossConnect.crossConnect, false, vxc.workspace)
		if err != nil {
			logrus.Error(err)
			return &empty.Empty{}, err
		}
	}

	if vxc.GetNext() != nil {
		vxc.GetNext().Close(ctx, conn)
	}

	return &empty.Empty{}, nil
}

// GetOpaque will return the corresponding outgoing connection
func (vxc *vppAgentXConnComposite) GetOpaque(incoming interface{}) interface{} {

	incomingConnection := incoming.(*connection.Connection)
	if crossConnect, ok := vxc.crossConnects[incomingConnection.GetId()]; ok {
		return crossConnect.ingressIfName
	}
	logrus.Errorf("GetOpaque outgoing not found for %v", incomingConnection)
	return nil
}

// NewVppAgentComposite creates a new VPP Agent composite
func newVppAgentXConnComposite(configuration *common.NSConfiguration) *vppAgentXConnComposite {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	logrus.Infof("newVppAgentComposite")

	newVppAgentXConnComposite := &vppAgentXConnComposite{
		vppAgentEndpoint: defaultVPPAgentEndpoint,
		crossConnects:    make(map[string]crossConnectStruct),
		workspace:        configuration.Workspace,
	}
	newVppAgentXConnComposite.reset()

	return newVppAgentXConnComposite
}

type vppAgentAclComposite struct {
	endpoint.BaseCompositeEndpoint
	vppAgentEndpoint string
	aclRules         map[string]string
}

func (vac *vppAgentAclComposite) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if vac.GetNext() == nil {
		logrus.Fatal("Should have Next set")
	}

	incoming, err := vac.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	opaque := vac.GetNext().GetOpaque(incoming)
	if opaque == nil {
		err := fmt.Errorf("Backend: Unable to find the ingressIfName")
		logrus.Errorf("%v", err)
		return nil, err
	}
	ingressIfName := opaque.(string)

	err = vac.applyAclOnVppInterface(ctx, "IngressACL", ingressIfName, vac.aclRules)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return incoming, nil
}

func (vac *vppAgentAclComposite) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if vac.GetNext() != nil {
		return vac.GetNext().Close(ctx, conn)
	}
	return &empty.Empty{}, nil
}

// NewVppAgentComposite creates a new VPP Agent composite
func newVppAgentAclComposite(configuration *common.NSConfiguration) *vppAgentAclComposite {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	logrus.Infof("newVppAgentComposite")

	newVppAgentAclComposite := &vppAgentAclComposite{
		vppAgentEndpoint: defaultVPPAgentEndpoint,
	}

	newVppAgentAclComposite.aclRules = getAclRulesConfig()

	return newVppAgentAclComposite
}
