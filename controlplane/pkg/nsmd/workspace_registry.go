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
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/sirupsen/logrus"
)

var workspaceRegistry = &WorkspaceRegistry{workspaceByEndpoint: make(map[string]*Workspace)}

type WorkspaceRegistry struct {
	workspaceByEndpoint map[string]*Workspace
	sync.Mutex
}

func WorkSpaceRegistry() *WorkspaceRegistry {
	return workspaceRegistry
}

func (w *WorkspaceRegistry) WorkspaceByEndpoint(endpoint *registry.NetworkServiceEndpoint) *Workspace {
	w.Lock()
	defer w.Unlock()
	return w.workspaceByEndpoint[endpoint.GetEndpointName()]
}

func (w *WorkspaceRegistry) AddEndpointToWorkspace(ws *Workspace, endpoint *registry.NetworkServiceEndpoint) {
	w.Lock()
	defer w.Unlock()
	logrus.Infof("w.workspaceByEndpoint[%s] = %v", endpoint.GetEndpointName(), ws)
	w.workspaceByEndpoint[endpoint.GetEndpointName()] = ws
}

func (w *WorkspaceRegistry) DeleteEndpointToWorkspace(endpointName string) {
	w.Lock()
	defer w.Unlock()
	delete(w.workspaceByEndpoint, endpointName)
}
