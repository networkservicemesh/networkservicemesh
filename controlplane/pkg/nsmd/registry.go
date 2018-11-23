// Copyright (c) 2018 Cisco and/or its affiliates.
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

package nsmd

import (
	"fmt"
	"github.com/ligato/networkservicemesh/controlplane/pkg/serviceregistry"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type registryServer struct {
	model           model.Model
	workspace       *Workspace
	serviceRegistry serviceregistry.ServiceRegistry
}

func NewRegistryServer(model model.Model, workspace *Workspace, serviceRegistry serviceregistry.ServiceRegistry) registry.NetworkServiceRegistryServer {
	return &registryServer{
		model:           model,
		workspace:       workspace,
		serviceRegistry: serviceRegistry,
	}
}

func (es *registryServer) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("Received RegisterNSE request: %v", request)

	// Check if there is already Network Service Endpoint object with the same name, if there is
	// success will be returned to NSE, since it is a case of NSE pod coming back up.
	client, err := es.serviceRegistry.RegistryClient()
	if err != nil {
		err = fmt.Errorf("attempt to connect to upstream registry failed with: %v", err)
		logrus.Error(err)
		return nil, err
	}

	// Some notes here:
	// 1)  Yes, we are overwriting anything we get for NetworkServiceManager
	//     from the NSE.  NSE's shouldn't specify NetworkServiceManager
	// 2)  We are not specifying Name or LastSeen, the nsmd-k8s will fill those
	//     in
	request.NetworkServiceManager = &registry.NetworkServiceManager{
		Url: es.serviceRegistry.GetPublicAPI(),
	}

	registration, err := client.RegisterNSE(context.Background(), request)
	if err != nil {
		err = fmt.Errorf("attempt to pass through from nsm to upstream registry failed with: %v", err)
		logrus.Error(err)
		return nil, err
	}

	ep := es.model.GetEndpoint(registration.GetNetworkserviceEndpoint().GetEndpointName())
	if ep == nil {
		es.model.AddEndpoint(registration)
		WorkSpaceRegistry().AddEndpointToWorkspace(es.workspace, registration.GetNetworkserviceEndpoint())
	}
	WorkSpaceRegistry().AddEndpointToWorkspace(es.workspace, ep.GetNetworkserviceEndpoint())
	logrus.Infof("Received upstream NSERegitration: %v", registration)

	return registration, nil
}

func (es *registryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	// TODO make sure we track which registry server we got the RegisterNSE from so we can only allow a deletion
	// of what you advertised
	logrus.Infof("Received Endpoint Remove request: %+v", request)
	client, err := es.serviceRegistry.RegistryClient()
	if err != nil {
		err = fmt.Errorf("attempt to pass through from nsm to upstream registry failed with: %v", err)
		logrus.Error(err)
		return nil, err
	}
	_, err = client.RemoveNSE(context.Background(), request)
	if err != nil {
		err = fmt.Errorf("attempt to pass through from nsm to upstream registry failed with: %v", err)
		logrus.Error(err)
		return nil, err
	}
	WorkSpaceRegistry().DeleteEndpointToWorkspace(request.EndpointName)
	if err := es.model.DeleteEndpoint(request.EndpointName); err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, nil
}

func (es *registryServer) Close() {

}
