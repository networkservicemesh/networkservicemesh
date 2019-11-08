// Copyright (c) 2019 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serviceregistryserver

import (
	"context"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/utils"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

const (
	// NSEExpirationTimeoutDefault - default Endpoint expiration timeout, NSE will be deleted if UpdateNetworkServiceEndpoint not received
	NSEExpirationTimeoutDefault = 5 * time.Minute
	// NSEExpirationTimeoutEnv - environment variable contains custom NSEExpirationTimeout
	NSEExpirationTimeoutEnv = utils.EnvVar("NSE_EXPIRATION_TIMEOUT")
)

// NSERegistryCache - cache of registered Network Service Endpoints
type NSERegistryCache interface {
	AddNetworkServiceEndpoint(nse *registry.NSERegistration) (*registry.NSERegistration, error)
	UpdateNetworkServiceEndpoint(nse *registry.NSERegistration) (*registry.NSERegistration, error)
	DeleteNetworkServiceEndpoint(endpointName string) (*registry.NSERegistration, error)
	GetEndpoints(networkServiceName string) []*registry.NSERegistration
	StartNSMDTracking(ctx context.Context)
}

type nseRegistryCache struct {
	sync.RWMutex
	networkServiceEndpoints map[string][]*registry.NSERegistration
	endpoints               map[string]*registry.NSERegistration
	nseExpirationTimeout    time.Duration
}

//NewNSERegistryCache creates new nerwork service endpoints cache
func NewNSERegistryCache() NSERegistryCache {
	return &nseRegistryCache{
		networkServiceEndpoints: make(map[string][]*registry.NSERegistration),
		endpoints:               make(map[string]*registry.NSERegistration),
		nseExpirationTimeout:    NSEExpirationTimeoutEnv.GetOrDefaultDuration(NSEExpirationTimeoutDefault),
	}
}

// AddNetworkServiceEndpoint - register NSE in cache
func (rc *nseRegistryCache) AddNetworkServiceEndpoint(entry *registry.NSERegistration) (*registry.NSERegistration, error) {
	rc.Lock()
	defer rc.Unlock()

	return rc.addNetworkServiceEndpoint(entry)
}

func (rc *nseRegistryCache) addNetworkServiceEndpoint(entry *registry.NSERegistration) (*registry.NSERegistration, error) {
	if endpoint, ok := rc.endpoints[entry.NetworkServiceEndpoint.Name]; ok {
		return nil, errors.Errorf("network service endpoint with name %s already exists: old: %v; new: %v", endpoint.NetworkServiceEndpoint.Name, endpoint, entry)
	}

	existingEndpoints := rc.GetEndpoints(entry.NetworkService.Name)
	for _, endpoint := range existingEndpoints {
		if !proto.Equal(endpoint.NetworkService, entry.NetworkService) {
			return nil, errors.Errorf("network service already exists with different parameters: old: %v; new: %v", endpoint, entry)
		}
	}

	entry.NetworkServiceManager.ExpirationTime = &timestamp.Timestamp{Seconds: time.Now().Add(rc.nseExpirationTimeout).Unix()}

	rc.networkServiceEndpoints[entry.NetworkService.Name] = append(rc.networkServiceEndpoints[entry.NetworkService.Name], entry)
	rc.endpoints[entry.NetworkServiceEndpoint.Name] = entry

	logrus.Infof("Registered NSE entry %v", entry)

	return entry, nil
}

func (rc *nseRegistryCache) UpdateNetworkServiceEndpoint(nse *registry.NSERegistration) (*registry.NSERegistration, error) {
	rc.Lock()
	defer rc.Unlock()

	if endpoint, ok := rc.endpoints[nse.NetworkServiceEndpoint.Name]; ok {
		if endpoint.NetworkServiceManager.Name != nse.NetworkServiceManager.Name {
			return nil, errors.Errorf("network service endpoint with name %s already registered from different NSM: old: %v; new: %v", endpoint.NetworkServiceEndpoint.Name, endpoint, nse)
		}
		endpoint.NetworkServiceManager.ExpirationTime = &timestamp.Timestamp{Seconds: time.Now().Add(rc.nseExpirationTimeout).Unix()}
		return endpoint, nil
	}

	return rc.addNetworkServiceEndpoint(nse)
}

// DeleteNetworkServiceEndpoint - remove NSE from cache
func (rc *nseRegistryCache) DeleteNetworkServiceEndpoint(endpointName string) (*registry.NSERegistration, error) {
	rc.Lock()
	defer rc.Unlock()

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
	return nil, errors.Errorf("endpoint %s not found", endpointName)
}

// GetEndpoints - get Endpoints list from cache by Name
func (rc *nseRegistryCache) GetEndpoints(networkServiceName string) []*registry.NSERegistration {
	return rc.networkServiceEndpoints[networkServiceName]
}

func (rc *nseRegistryCache) StartNSMDTracking(ctx context.Context) {
	span := spanhelper.FromContext(ctx, "NsmrsCache.StartNSMDTracking")
	defer span.Finish()
	logger := span.Logger()

	go func() {
		for {
			<-time.After(rc.nseExpirationTimeout / 2)
			for endpointName, endpoint := range rc.endpoints {
				if endpoint.NetworkServiceManager.ExpirationTime.Seconds < time.Now().Unix() {
					nse, err := rc.DeleteNetworkServiceEndpoint(endpointName)
					if err != nil {
						logger.Errorf("Unexpected registry error : %v", err)
					}
					logger.Infof("Network Service Endpoint removed by timeout : %v", nse)
				}
			}
		}
	}()
	logger.Infof("NSMD tracking started")
}
