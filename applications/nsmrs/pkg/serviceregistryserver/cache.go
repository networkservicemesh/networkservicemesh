package serviceregistryserver

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type NSERegistryCache interface {
	AddNetworkServiceEndpoint(nse *registry.NSERegistration) (*registry.NSERegistration, error)
	DeleteNetworkServiceEndpoint(endpointName string) (*registry.NSERegistration, error)
	GetEndpointsByNs(networkServiceName string) []*registry.NSERegistration
}

type nseRegistryCache struct {
	networkServiceEndpoints map[string][]*registry.NSERegistration
}

//NewNSERegistryCache creates new nerwork service server registry cache
func NewNSERegistryCache() *nseRegistryCache {
	return &nseRegistryCache{
		networkServiceEndpoints: make(map[string][]*registry.NSERegistration),
	}
}

// AddNetworkServiceEndpoint - register NSE in cache
func (rc *nseRegistryCache) AddNetworkServiceEndpoint(nse *registry.NSERegistration) (*registry.NSERegistration, error) {
	existingEndpoints := rc.GetEndpointsByNs(nse.NetworkService.Name)

	for _, endpoint := range existingEndpoints {
		if !proto.Equal(endpoint.NetworkService, nse.NetworkService) {
			return nil, fmt.Errorf("network service already exists with different parameters: old: %v; new: %v", endpoint, nse)
		}
	}

	rc.networkServiceEndpoints[nse.NetworkService.Name] = append(rc.networkServiceEndpoints[nse.NetworkService.Name], nse)
	return nse, nil
}

// DeleteNetworkServiceEndpoint - remove NSE from cache
func (rc *nseRegistryCache) DeleteNetworkServiceEndpoint(endpointName string) (*registry.NSERegistration, error) {
	for networkService, endpointList := range rc.networkServiceEndpoints {
		for i := range endpointList {
			if endpointList[i].NetworkServiceEndpoint.Name == endpointName {
				endpoint := endpointList[i]
				rc.networkServiceEndpoints[networkService] = append(endpointList[:i], endpointList[i+1:]...)
				return endpoint, nil
			}
		}
	}
	return nil, fmt.Errorf("endpoint %s not found", endpointName)
}

// GetEndpointsByNs - get Endpoints list from cache by Name
func (rc *nseRegistryCache) GetEndpointsByNs(networkServiceName string) []*registry.NSERegistration {
	return rc.networkServiceEndpoints[networkServiceName]
}
