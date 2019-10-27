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

package remote

import (
	"github.com/networkservicemesh/networkservicemesh/sdk/compat"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/remote"

	remotenetworkservice "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
)

// NewRemoteNetworkServiceServer -  creates a new remote.NetworkServiceServer
func NewRemoteNetworkServiceServer(manager nsm.NetworkServiceManager, connectionMonitor remote.MonitorServer) remotenetworkservice.NetworkServiceServer {
	return compat.NewUnifiedNetworkServiceServerAdapter(NewCompositeService(
		NewRequestValidator(),
		NewMonitorService(connectionMonitor),
		NewConnectionService(manager.Model()),
		NewForwarderService(manager.Model(), manager.ServiceRegistry()),
		NewEndpointSelectorService(manager.NseManager(), manager.PluginRegistry(), manager.Model()),
		NewEndpointService(manager.NseManager(), manager.GetHealProperties(), manager.Model(), manager.PluginRegistry()),
		NewCrossConnectService(),
	), nil)
}
