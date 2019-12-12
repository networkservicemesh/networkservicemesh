package nsmd

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/local"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

// NewNetworkServiceServer - construct a local network service chain
func NewNetworkServiceServer(model model.Model, ws *Workspace,
	nsmManager nsm.NetworkServiceManager) networkservice.NetworkServiceServer {
	return common.NewCompositeService("Local",
		common.NewRequestValidator(),
		common.NewMonitorService(ws.MonitorConnectionServer()),
		local.NewWorkspaceService(ws.Name()),
		local.NewConnectionService(model),
		common.NewForwarderService(model, nsmManager.ServiceRegistry(), common.NewLocalMechanismSelector()),
		local.NewEndpointSelectorService(nsmManager.NseManager()),
		common.NewExcludedPrefixesService(),
		local.NewEndpointService(nsmManager.NseManager(), nsmManager.GetHealProperties(), nsmManager.Model()),
		common.NewCrossConnectService(),
	)
}
