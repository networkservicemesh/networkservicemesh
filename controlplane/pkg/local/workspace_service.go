// Copyright (c) 2019 Cisco and/or its affiliates.
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

package local

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	mechanismCommon "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type workspaceProviderService struct {
	name string
}

// NewWorkspaceService - creates a service to update workspace information for request
func NewWorkspaceService(name string) networkservice.NetworkServiceServer {
	return &workspaceProviderService{
		name: name,
	}
}

func (srv *workspaceProviderService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logrus.Infof("Received request from client to connect to NetworkService: %v", request)
	srv.updateMechanisms(request)

	ctx = common.WithWorkspaceName(ctx, srv.name)
	result, err := common.ProcessNext(ctx, request)
	if result != nil {
		// Remove workspace field since clients doesn't require them.
		delete(result.GetMechanism().GetParameters(), mechanismCommon.Workspace)
	}

	return result, err
}

func (srv *workspaceProviderService) updateMechanisms(request *networkservice.NetworkServiceRequest) {
	// Update passed local mechanism parameters to contains a workspace name
	for _, mechanism := range request.MechanismPreferences {
		if mechanism.Parameters == nil {
			mechanism.Parameters = map[string]string{}
		}
		mechanism.Parameters[mechanismCommon.Workspace] = srv.name
	}
}

func (srv *workspaceProviderService) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	ctx = common.WithWorkspaceName(ctx, srv.name)
	return common.ProcessClose(ctx, connection)
}
