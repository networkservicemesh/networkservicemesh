package serviceregistryserver

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type discoveryService struct {
	cache NSERegistryCache
}

func newDiscoveryService(cache NSERegistryCache) *discoveryService {
	return &discoveryService{
		cache: cache,
	}
}

func (d *discoveryService) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	networkServiceEnpoints := d.cache.GetEndpointsByNs(request.NetworkServiceName)
	if len(networkServiceEnpoints) == 0 {
		err := fmt.Errorf("no NetworkService with name: %v", request.NetworkServiceName)
		logrus.Errorf("Cannot find Network Service: %v", err)
		return nil, err
	}

	response := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name:    request.NetworkServiceName,
			Payload: networkServiceEnpoints[0].Nse.NetworkService.Payload,
			Matches: networkServiceEnpoints[0].Nse.NetworkService.Matches,
		},
		NetworkServiceManagers: make(map[string]*registry.NetworkServiceManager),
		Payload:                networkServiceEnpoints[0].Nse.NetworkService.Payload,
	}

	for _, endpoint := range networkServiceEnpoints {
		response.NetworkServiceManagers[endpoint.Nse.NetworkServiceManager.Name] = endpoint.Nse.NetworkServiceManager
		response.NetworkServiceEndpoints = append(response.NetworkServiceEndpoints, endpoint.Nse.NetworkServiceEndpoint)
	}

	logrus.Infof("FindNetworkService done: %v", response)

	return response, nil
}
