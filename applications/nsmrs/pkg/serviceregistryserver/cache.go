package serviceregistryserver

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type NSERegistryCache interface {
	AddNetworkServiceEndpoint(nse *NSECacheEntry) (*NSECacheEntry, error)
	DeleteNetworkServiceEndpoint(endpointName string) (*NSECacheEntry, error)
	GetEndpointsByNs(networkServiceName string) []*NSECacheEntry
}

type NSECacheEntry struct {
	nse *registry.NSERegistration
	monitor *nsmMonitor
}

type nseRegistryCache struct {
	networkServiceEndpoints map[string][]*NSECacheEntry
}

//NewNSERegistryCache creates new nerwork service server registry cache
func NewNSERegistryCache() *nseRegistryCache {
	return &nseRegistryCache{
		networkServiceEndpoints: make(map[string][]*NSECacheEntry),
	}
}

// AddNetworkServiceEndpoint - register NSE in cache
func (rc *nseRegistryCache) AddNetworkServiceEndpoint(entry *NSECacheEntry) (*NSECacheEntry, error) {
	existingEndpoints := rc.GetEndpointsByNs(entry.nse.NetworkService.Name)

	for _, endpoint := range existingEndpoints {
		if !proto.Equal(endpoint.nse.NetworkService, entry.nse.NetworkService) {
			return nil, fmt.Errorf("network service already exists with different parameters: old: %v; new: %v", endpoint, entry)
		}
	}

	rc.networkServiceEndpoints[entry.nse.NetworkService.Name] = append(rc.networkServiceEndpoints[entry.nse.NetworkService.Name], entry)
	return entry, nil
}

// DeleteNetworkServiceEndpoint - remove NSE from cache
func (rc *nseRegistryCache) DeleteNetworkServiceEndpoint(endpointName string) (*NSECacheEntry, error) {
	for networkService, endpointList := range rc.networkServiceEndpoints {
		for i := range endpointList {
			if endpointList[i].nse.NetworkServiceEndpoint.Name == endpointName {
				endpoint := endpointList[i]
				rc.networkServiceEndpoints[networkService] = append(endpointList[:i], endpointList[i+1:]...)
				return endpoint, nil
			}
		}
	}
	return nil, fmt.Errorf("endpoint %s not found", endpointName)
}

// GetEndpointsByNs - get Endpoints list from cache by Name
func (rc *nseRegistryCache) GetEndpointsByNs(networkServiceName string) []*NSECacheEntry {
	return rc.networkServiceEndpoints[networkServiceName]
}
