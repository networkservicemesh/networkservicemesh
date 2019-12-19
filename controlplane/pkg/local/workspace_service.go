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

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	mechanismCommon "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
)

// WorkspaceProvider provides workspace search function
type WorkspaceProvider interface {
	WorkspaceNameByGRPCContext(ctx context.Context) string
}

type workspaceProviderService struct {
	workspaceProvider WorkspaceProvider
}

// NewWorkspaceService - creates a service to update workspace information for request
func NewWorkspaceService(workspaceProvider WorkspaceProvider) networkservice.NetworkServiceServer {
	return &workspaceProviderService{
		workspaceProvider: workspaceProvider,
	}
}

func (srv *workspaceProviderService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	workspace := srv.workspaceProvider.WorkspaceNameByGRPCContext(ctx)
	logrus.Infof("Workspace service request: client: %s, request: %v", workspace, request)
	srv.updateMechanisms(request, workspace)

	ctx = common.WithWorkspaceName(ctx, workspace)
	result, err := common.ProcessNext(ctx, request)
	if result != nil {
		// Remove workspace field since clients doesn't require them.
		delete(result.GetMechanism().GetParameters(), mechanismCommon.Workspace)
	}

	return result, err
}

func (srv *workspaceProviderService) updateMechanisms(request *networkservice.NetworkServiceRequest, workspace string) {
	// Update passed local mechanism parameters to contains a workspace name
	for _, mechanism := range request.MechanismPreferences {
		if mechanism.Parameters == nil {
			mechanism.Parameters = map[string]string{}
		}
		mechanism.Parameters[mechanismCommon.Workspace] = workspace
	}
}

func (srv *workspaceProviderService) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	workspace := srv.workspaceProvider.WorkspaceNameByGRPCContext(ctx)
	ctx = common.WithWorkspaceName(ctx, workspace)
	return common.ProcessClose(ctx, connection)
}
