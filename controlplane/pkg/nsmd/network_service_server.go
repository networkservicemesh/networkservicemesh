package nsmd

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/local"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"
)

// NewNetworkServiceServer - construct a local network service chain
func NewNetworkServiceServer(model model.Model, connectionMonitorServer connectionmonitor.MonitorServer,
	nsmManager nsm.NetworkServiceManager, workspaceProvider local.WorkspaceProvider) networkservice.NetworkServiceServer {
	return common.NewCompositeService("Local",
		common.NewRequestValidator(),
		common.NewMonitorService(connectionMonitorServer),
		local.NewWorkspaceService(workspaceProvider),
		local.NewConnectionService(model),
		local.NewForwarderService(model, nsmManager.ServiceRegistry()),
		local.NewEndpointSelectorService(nsmManager.NseManager()),
		common.NewExcludedPrefixesService(),
		local.NewEndpointService(nsmManager.NseManager(), nsmManager.GetHealProperties(), nsmManager.Model()),
		common.NewCrossConnectService(),
	)
}
