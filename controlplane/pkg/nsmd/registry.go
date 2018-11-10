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

	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type registryServer struct {
	model model.Model
}

func NewRegistryServer(model model.Model) registry.NetworkServiceRegistryServer {
	return &registryServer{
		model: model,
	}
}

func (es *registryServer) RegisterNSE(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	logrus.Infof("Received RegisterNSE request: %v", request)

	// Check if there is already Network Service Endpoint object with the same name, if there is
	// success will be returned to NSE, since it is a case of NSE pod coming back up.
	client, err := RegistryClient()
	if err != nil {
		err = fmt.Errorf("Attempt to pass through from nsm to upstream registry failed with:", err)
		logrus.Error(err)
		return nil, err
	}
	endpoint, err := client.RegisterNSE(context.Background(), request)
	if err != nil {
		err = fmt.Errorf("Attempt to pass through from nsm to upstream registry failed with:", err)
		logrus.Error(err)
		return nil, err
	}

	ep := es.model.GetEndpoint(endpoint.EndpointName)
	if ep == nil {
		es.model.AddEndpoint(endpoint)
	}

	return endpoint, nil
}

func (es *registryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*common.Empty, error) {
	// TODO make sure we track which registry server we got the RegisterNSE from so we can only allow a deletion
	// of what you advertised
	logrus.Infof("Received Endpoint Remove request: %+v", request)
	client, err := RegistryClient()
	if err != nil {
		err = fmt.Errorf("Attempt to pass through from nsm to upstream registry failed with:", err)
		logrus.Error(err)
		return nil, err
	}
	_, err = client.RemoveNSE(context.Background(), request)
	if err != nil {
		err = fmt.Errorf("Attempt to pass through from nsm to upstream registry failed with:", err)
		logrus.Error(err)
		return nil, err
	}
	if err := es.model.DeleteEndpoint(request.EndpointName); err != nil {
		return &common.Empty{}, err
	}
	return &common.Empty{}, nil
}

func (es *registryServer) Close() {

}
