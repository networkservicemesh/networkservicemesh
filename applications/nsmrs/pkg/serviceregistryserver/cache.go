package serviceregistryserver

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

const (
	NSEExpirationTimeout  = 5 * time.Minute
	NSEValidationInterval = 2 * time.Minute
)

// NSERegistryCache - cache of registered Network Service Endpoints
type NSERegistryCache interface {
	AddNetworkServiceEndpoint(nse *registry.NSERegistration) (*registry.NSERegistration, error)
	UpdateNetworkServiceEndpoint(nse *registry.NSERegistration) (*registry.NSERegistration, error)
	DeleteNetworkServiceEndpoint(endpointName string) (*registry.NSERegistration, error)
	GetEndpointsByNs(networkServiceName string) []*registry.NSERegistration
	StartNSMDTracking()
}

type nseRegistryCache struct {
	networkServiceEndpoints map[string][]*registry.NSERegistration
	endpoints               map[string]*registry.NSERegistration
}

//NewNSERegistryCache creates new nerwork service endpoints cache
func NewNSERegistryCache() NSERegistryCache {
	return &nseRegistryCache{
		networkServiceEndpoints: make(map[string][]*registry.NSERegistration),
		endpoints:               make(map[string]*registry.NSERegistration),
	}
}

// AddNetworkServiceEndpoint - register NSE in cache
func (rc *nseRegistryCache) AddNetworkServiceEndpoint(entry *registry.NSERegistration) (*registry.NSERegistration, error) {
	if endpoint, ok := rc.endpoints[entry.NetworkServiceEndpoint.Name]; ok {
		return nil, fmt.Errorf("network service endpoint with name %s already exists: old: %v; new: %v", endpoint.NetworkServiceEndpoint.Name, endpoint, entry)
	}

	existingEndpoints := rc.GetEndpointsByNs(entry.NetworkService.Name)
	for _, endpoint := range existingEndpoints {
		if !proto.Equal(endpoint.NetworkService, entry.NetworkService) {
			return nil, fmt.Errorf("network service already exists with different parameters: old: %v; new: %v", endpoint, entry)
		}
	}

	entry.NetworkServiceManager.ExpirationTime = &timestamp.Timestamp{Seconds: time.Now().Add(NSEExpirationTimeout).Unix()}

	rc.networkServiceEndpoints[entry.NetworkService.Name] = append(rc.networkServiceEndpoints[entry.NetworkService.Name], entry)
	rc.endpoints[entry.NetworkServiceEndpoint.Name] = entry

	logrus.Infof("Registered NSE entry %v", entry)

	return entry, nil
}

func (rc *nseRegistryCache) UpdateNetworkServiceEndpoint(nse *registry.NSERegistration) (*registry.NSERegistration, error) {
	if endpoint, ok := rc.endpoints[nse.NetworkServiceEndpoint.Name]; ok {
		if endpoint.NetworkServiceManager.Name != nse.NetworkServiceManager.Name {
			return nil, fmt.Errorf("network service endpoint with name %s already registered from different NSM: old: %v; new: %v", endpoint.NetworkServiceEndpoint.Name, endpoint, nse)
		}
		endpoint.NetworkServiceManager.ExpirationTime = &timestamp.Timestamp{Seconds: time.Now().Add(NSEExpirationTimeout).Unix()}
		return endpoint, nil
	}

	return rc.AddNetworkServiceEndpoint(nse)
}

// DeleteNetworkServiceEndpoint - remove NSE from cache
func (rc *nseRegistryCache) DeleteNetworkServiceEndpoint(endpointName string) (*registry.NSERegistration, error) {
	delete(rc.endpoints, endpointName)
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

func (rc *nseRegistryCache) StartNSMDTracking() {
	go func() {
		for {
			<-time.After(NSEValidationInterval)
			for endpointName, endpoint := range rc.endpoints {
				logrus.Info("COMPARE %v %v", endpoint.NetworkServiceManager.ExpirationTime.Seconds, time.Now().Unix())
				if endpoint.NetworkServiceManager.ExpirationTime.Seconds < time.Now().Unix() {
					_, err := rc.DeleteNetworkServiceEndpoint(endpointName)
					if err != nil {
						logrus.Errorf("Unexpected registry error : %v", err)
					}
				}
			}
		}
	}()
	logrus.Infof("NSMD tracking started")
}
