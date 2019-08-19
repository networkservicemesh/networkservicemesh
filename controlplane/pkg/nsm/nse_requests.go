package nsm

import (
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

func (srv *networkServiceManager) createRemoteNSMRequest(endpoint *registry.NSERegistration, requestConn connection.Connection, dp *model.Dataplane, existingCC *model.ClientConnection) networkservice.Request {
	// We need to obtain parameters for remote mechanism
	remoteM := append([]connection.Mechanism{}, dp.RemoteMechanisms...)

	// Try Heal only if endpoint are same as for existing connection.
	if existingCC != nil && endpoint == existingCC.Endpoint {
		if remoteDst := existingCC.Xcon.GetRemoteDestination(); remoteDst != nil {
			return remote_networkservice.NewRequest(
				&remote_connection.Connection{
					Id:                                   remoteDst.GetId(),
					NetworkService:                       remoteDst.NetworkService,
					Context:                              remoteDst.GetContext(),
					Labels:                               remoteDst.GetLabels(),
					DestinationNetworkServiceManagerName: endpoint.GetNetworkServiceManager().GetName(),
					SourceNetworkServiceManagerName:      srv.getNetworkServiceManagerName(),
					NetworkServiceEndpointName:           endpoint.GetNetworkServiceEndpoint().GetName(),
				},
				remoteM,
			)
		}
	}

	return remote_networkservice.NewRequest(
		&remote_connection.Connection{
			Id:                                   "-",
			NetworkService:                       requestConn.GetNetworkService(),
			Context:                              requestConn.GetContext(),
			Labels:                               requestConn.GetLabels(),
			DestinationNetworkServiceManagerName: endpoint.GetNetworkServiceManager().GetName(),
			SourceNetworkServiceManagerName:      srv.getNetworkServiceManagerName(),
			NetworkServiceEndpointName:           endpoint.GetNetworkServiceEndpoint().GetName(),
		},
		remoteM,
	)
}

func (srv *networkServiceManager) createLocalNSERequest(endpoint *registry.NSERegistration, dp *model.Dataplane, requestConn connection.Connection) networkservice.Request {
	// We need to obtain parameters for local mechanism
	localM := append([]connection.Mechanism{}, dp.LocalMechanisms...)

	return local_networkservice.NewRequest(
		&local_connection.Connection{
			Id:             srv.createConnectionId(),
			NetworkService: endpoint.GetNetworkService().GetName(),
			Context:        requestConn.GetContext(),
			Labels:         requestConn.GetLabels(),
		},
		localM,
	)
}
