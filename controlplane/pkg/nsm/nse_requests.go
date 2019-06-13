package nsm

import (
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

func (srv *networkServiceManager) createRemoteNSMRequest(endpoint *registry.NSERegistration, requestConnection connection.Connection, dataplane *model.Dataplane, existingConnection *model.ClientConnection) (*remote_networkservice.NetworkServiceRequest, error) {
	// We need to obtain parameters for remote mechanism
	remoteM := append([]connection.Mechanism{}, dataplane.RemoteMechanisms...)

	// Try Heal only if endpoint are same as for existing connection.
	if existingConnection != nil && endpoint == existingConnection.Endpoint {
		if remoteDst := existingConnection.Xcon.GetRemoteDestination(); remoteDst != nil {
			return remote_networkservice.NewRequest(
				&remote_connection.Connection{
					Id:                                   remoteDst.GetId(),
					NetworkService:                       remoteDst.NetworkService,
					Context:                              remoteDst.GetContext(),
					Labels:                               remoteDst.GetLabels(),
					DestinationNetworkServiceManagerName: endpoint.GetNetworkServiceManager().GetName(),
					SourceNetworkServiceManagerName:      srv.getNetworkServiceManagerName(),
					NetworkServiceEndpointName:           endpoint.GetNetworkserviceEndpoint().GetEndpointName(),
				},
				remoteM,
			), nil
		}
	}

	return remote_networkservice.NewRequest(
		&remote_connection.Connection{
			Id:                                   "-",
			NetworkService:                       requestConnection.GetNetworkService(),
			Context:                              requestConnection.GetContext(),
			Labels:                               requestConnection.GetLabels(),
			DestinationNetworkServiceManagerName: endpoint.GetNetworkServiceManager().GetName(),
			SourceNetworkServiceManagerName:      srv.getNetworkServiceManagerName(),
			NetworkServiceEndpointName:           endpoint.GetNetworkserviceEndpoint().GetEndpointName(),
		},
		remoteM,
	), nil
}

func (srv *networkServiceManager) createLocalNSERequest(endpoint *registry.NSERegistration, dataplane *model.Dataplane, requestConnection connection.Connection) *local_networkservice.NetworkServiceRequest {
	// We need to obtain parameters for local mechanism
	localM := append([]connection.Mechanism{}, dataplane.LocalMechanisms...)

	return local_networkservice.NewRequest(
		&local_connection.Connection{
			Id:             srv.createConnectionId(),
			NetworkService: endpoint.GetNetworkService().GetName(),
			Context:        requestConnection.GetContext(),
			Labels:         requestConnection.GetLabels(),
		},
		localM,
	)
}
