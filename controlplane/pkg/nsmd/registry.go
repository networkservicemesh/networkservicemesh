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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type NSERegistryServer interface {
	registry.NetworkServiceRegistryServer
	RegisterNSEWithClient(ctx context.Context, request *registry.NSERegistration, client registry.NetworkServiceRegistryClient) (*registry.NSERegistration, error)
}
type registryServer struct {
	nsm       *nsmServer
	workspace *Workspace
}

func NewRegistryServer(nsm *nsmServer, workspace *Workspace) NSERegistryServer {
	return &registryServer{
		nsm:       nsm,
		workspace: workspace,
	}
}

func (es *registryServer) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("Received RegisterNSE request: %v", request)

	// Check if there is already Network Service Endpoint object with the same name, if there is
	// success will be returned to NSE, since it is a case of NSE pod coming back up.
	client, err := es.nsm.serviceRegistry.NseRegistryClient(context.Background())
	if err != nil {
		err = errors.Wrap(err, "attempt to connect to upstream registry failed with")
		logrus.Error(err)
		return nil, err
	}

	reg, err := es.RegisterNSEWithClient(ctx, request, client)
	if err != nil {
		return reg, err
	}

	// Append to workspace...
	err = es.workspace.localRegistry.AppendNSERegRequest(es.workspace.name, reg)
	if err != nil {
		logrus.Errorf("Failed to store NSE into local registry service: %v", err)
		_, _ = client.RemoveNSE(context.Background(), &registry.RemoveNSERequest{NetworkServiceEndpointName: reg.GetNetworkServiceEndpoint().GetName()})
		return nil, err
	}
	return reg, nil
}
func (es *registryServer) RegisterNSEWithClient(ctx context.Context, request *registry.NSERegistration, client registry.NetworkServiceRegistryClient) (*registry.NSERegistration, error) {
	// Some notes here:
	// 1)  Yes, we are overwriting anything we get for NetworkServiceManager
	//     from the NSE.  NSE's shouldn't specify NetworkServiceManager
	// 2)  We are not specifying Name or LastSeen, the nsmd-k8s will fill those
	//     in
	request.NetworkServiceManager = &registry.NetworkServiceManager{
		Url: es.nsm.serviceRegistry.GetPublicAPI(),
	}

	registration, err := client.RegisterNSE(context.Background(), request)
	if err != nil {
		err = errors.Wrap(err, "attempt to pass through from nsm to upstream registry failed with")
		logrus.Error(err)
		return nil, err
	}

	ep := es.nsm.model.GetEndpoint(registration.GetNetworkServiceEndpoint().GetName())
	modelEndpoint := &model.Endpoint{
		SocketLocation: es.workspace.NsmClientSocket(),
		Endpoint:       registration,
		Workspace:      es.workspace.Name(),
	}
	if ep == nil {
		es.nsm.model.AddEndpoint(ctx, modelEndpoint)
	}
	logrus.Infof("Received upstream NSERegitration: %v", registration)

	return registration, nil
}

func (es *registryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	// TODO make sure we track which registry server we got the RegisterNSE from so we can only allow a deletion
	// of what you advertised
	logrus.Infof("Received Endpoint Remove request: %+v", request)
	client, err := es.nsm.serviceRegistry.NseRegistryClient(context.Background())
	if err != nil {
		err = errors.Wrap(err, "attempt to pass through from nsm to upstream registry failed with")
		logrus.Error(err)
		return nil, err
	}
	_, err = client.RemoveNSE(context.Background(), request)
	if err != nil {
		err = errors.Wrap(err, "attempt to pass through from nsm to upstream registry failed")
		logrus.Error(err)
		return nil, err
	}
	es.nsm.model.DeleteEndpoint(context.Background(), request.GetNetworkServiceEndpointName())
	return &empty.Empty{}, nil
}

func (es *registryServer) Close() {

}
