package nsm

import (
	"github.com/golang/protobuf/proto"

	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	local_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
)

func cloneConnection(request nsm.NSMRequest, response nsm.NSMConnection) nsm.NSMConnection {
	if request.IsRemote() {
		return proto.Clone(response.(*remote_connection.Connection)).(*remote_connection.Connection)
	} else {
		return proto.Clone(response.(*local_connection.Connection)).(*local_connection.Connection)
	}
}

func newConnection(request nsm.NSMRequest) nsm.NSMConnection {
	if request.IsRemote() {
		return proto.Clone(request.(*remote_networkservice.NetworkServiceRequest).Connection).(*remote_connection.Connection)
	} else {
		return proto.Clone(request.(*local_networkservice.NetworkServiceRequest).Connection).(*local_connection.Connection)
	}
}

func filterEndpoints(endpoints []*registry.NetworkServiceEndpoint, ignore_endpoints map[string]*registry.NSERegistration) []*registry.NetworkServiceEndpoint {
	result := []*registry.NetworkServiceEndpoint{}
	// Do filter of endpoints
	for _, candidate := range endpoints {
		if ignore_endpoints[candidate.GetEndpointName()] == nil {
			result = append(result, candidate)
		}
	}
	return result
}

func filterRegEndpoints(endpoints []*registry.NSERegistration, ignore_endpoints map[string]*registry.NSERegistration) []*registry.NSERegistration {
	result := []*registry.NSERegistration{}
	// Do filter of endpoints
	for _, candidate := range endpoints {
		if ignore_endpoints[candidate.GetNetworkserviceEndpoint().GetEndpointName()] == nil {
			result = append(result, candidate)
		}
	}
	return result
}
