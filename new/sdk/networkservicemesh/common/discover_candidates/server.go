package discover_candidates

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
)

type discoverCandidatesServer struct {
	registry registry.NetworkServiceDiscoveryClient
}

func NewServer(registry registry.NetworkServiceDiscoveryClient) networkservice.NetworkServiceServer {
	return &discoverCandidatesServer{registry: registry}
}

func (d *discoverCandidatesServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	// TODO - handle case where NetworkServiceEndpoint is already set
	if request.GetConnection().GetNetworkServiceEndpointName() != "" {
		// TODO what to do in this case?
	}
	registryRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: request.GetConnection().GetNetworkService(),
	}
	registryResponse, err := d.registry.FindNetworkService(ctx, registryRequest)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	registryResponse.NetworkServiceEndpoints = matchEndpoint(request.GetConnection().GetLabels(), registryResponse.GetNetworkService(), registryResponse.GetNetworkServiceEndpoints())
	// TODO handle local case
	ctx = WithCandidates(ctx, registryResponse)
	return next.Server(ctx).Request(ctx, request)
}

func (d *discoverCandidatesServer) Close(context.Context, *connection.Connection) (*empty.Empty, error) {
	panic("implement me")
}
