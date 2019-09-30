package serviceregistryserver

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

// NSERegistryCache - cache of registered Network Service Endpoints
type NSERegistryCache interface {
	AddNetworkServiceEndpoint(nse *NSECacheEntry) (*NSECacheEntry, error)
	DeleteNetworkServiceEndpoint(endpointName string) (*NSECacheEntry, error)
	GetEndpointsByNs(networkServiceName string) []*NSECacheEntry
}

// NSECacheEntry - entry of NSERegistryCache, contains information about NSE and NSMgr monitor
type NSECacheEntry struct {
	Nse     *registry.NSERegistration
	Monitor NsmMonitor
}

type nseRegistryCache struct {
	networkServiceEndpoints map[string][]*NSECacheEntry
}

//NewNSERegistryCache creates new nerwork service endpoints cache
func NewNSERegistryCache() NSERegistryCache {
	return &nseRegistryCache{
		networkServiceEndpoints: make(map[string][]*NSECacheEntry),
	}
}

// AddNetworkServiceEndpoint - register NSE in cache
func (rc *nseRegistryCache) AddNetworkServiceEndpoint(entry *NSECacheEntry) (*NSECacheEntry, error) {
	for _, endpoints := range rc.networkServiceEndpoints {
		for _, endpoint := range endpoints {
			if endpoint.Nse.NetworkServiceEndpoint.Name == entry.Nse.NetworkServiceEndpoint.Name {
				return nil, fmt.Errorf("network service endpoint with name %s already exists: old: %v; new: %v", endpoint.Nse.NetworkServiceEndpoint.Name, endpoint, entry)
			}
		}
	}

	existingEndpoints := rc.GetEndpointsByNs(entry.Nse.NetworkService.Name)
	for _, endpoint := range existingEndpoints {
		if !proto.Equal(endpoint.Nse.NetworkService, entry.Nse.NetworkService) {
			return nil, fmt.Errorf("network service already exists with different parameters: old: %v; new: %v", endpoint, entry)
		}
	}

	rc.networkServiceEndpoints[entry.Nse.NetworkService.Name] = append(rc.networkServiceEndpoints[entry.Nse.NetworkService.Name], entry)
	return entry, nil
}

// DeleteNetworkServiceEndpoint - remove NSE from cache
func (rc *nseRegistryCache) DeleteNetworkServiceEndpoint(endpointName string) (*NSECacheEntry, error) {
	for networkService, endpointList := range rc.networkServiceEndpoints {
		for i := range endpointList {
			if endpointList[i].Nse.NetworkServiceEndpoint.Name == endpointName {
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
