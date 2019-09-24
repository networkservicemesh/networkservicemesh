package serviceregistryserver

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/sirupsen/logrus"
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
	if networkServiceEnpoints == nil || len(networkServiceEnpoints) == 0 {
		err := fmt.Errorf("no NetworkService with name: %v", request.NetworkServiceName)
		logrus.Errorf("Cannot file Network Service: %v", err)
		return nil, err
	}

	response := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name: request.NetworkServiceName,
			Payload: networkServiceEnpoints[0].NetworkService.Payload,
			Matches: networkServiceEnpoints[0].NetworkService.Matches,
		},
		NetworkServiceManagers: make(map[string]*registry.NetworkServiceManager),
		Payload: networkServiceEnpoints[0].NetworkService.Payload,
	}

	for _, endpoint := range networkServiceEnpoints {
		response.NetworkServiceManagers[endpoint.NetworkServiceManager.Name] = endpoint.NetworkServiceManager
		response.NetworkServiceEndpoints = append(response.NetworkServiceEndpoints, endpoint.NetworkServiceEndpoint)
	}

	logrus.Infof("FindNetworkService done: %v", response)

	return response, nil
}
