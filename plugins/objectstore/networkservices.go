// Copyright (c) 2018 Cisco and/or its affiliates.
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

package objectstore

import (
	"sync"

	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	// "github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
)

// NetworkServicesStore map stores all discovered Network Service Object
// with a key composed of a name and a namespace
type networkServicesStore struct {
	networkService map[string]*v1.NetworkService
	sync.RWMutex
}

// NewNetworkServicesStore instantiates a new instance of a global
// NetworkServices store. It must be initialized before any controllers start.
func newNetworkServicesStore() *networkServicesStore {
	return &networkServicesStore{
		networkService: map[string]*v1.NetworkService{}}
}

// Add method adds descovered NetworkService if it does not
// already exit in the store.
func (n *networkServicesStore) Add(ns *v1.NetworkService) {
	n.Lock()
	defer n.Unlock()
	/*
		key := meta{
			name:      ns.ObjectMeta.Name,
			namespace: ns.ObjectMeta.Namespace,
		}
	*/
	if _, ok := n.networkService[ns.ObjectMeta.Name]; !ok {
		// Not in the store, adding it.
		n.networkService[ns.ObjectMeta.Name] = ns
	}
}

// Get method returns NetworkService, if it does not
// already it returns nil.
func (n *networkServicesStore) Get(nsName string) *v1.NetworkService {
	n.Lock()
	defer n.Unlock()
	/*
		key := meta{
			name:      nsName,
			namespace: nsNamespace,
		}
	*/
	ns, ok := n.networkService[nsName]
	if !ok {
		return nil
	}
	return ns
}

// Get method returns NetworkService, if it does not
// already it returns nil.
/* func (n *networkServicesStore) AddEndpointToNetworkService(nsName string, nsNamespace string, ch *netmesh.NetworkServiceEndpoint) error {
	n.Lock()
	defer n.Unlock()

	key := meta{
		name:      nsName,
		namespace: nsNamespace,
	}
	ns, ok := n.networkService[key]
	if !ok {
		return fmt.Errorf("failed to find network service %s/%s in the object store", key.namespace, key.name)
	}

	// Need to check if NetworkService has already Endpoint with the same name and namespace, Endpoints must
	// be unique for Name and Namspace pair.
	found := false
	for _, c := range ns.Endpoint {
		if c.Metadata.Name == ch.Metadata.Name && c.Metadata.Namespace == ch.Metadata.Namespace {
			found = true
			break
		}
	}
	if found {
		return fmt.Errorf("failed to add Endpoint %s/%s to network service %s/%s, the Endpoint already exists in the object store",
			ch.Metadata.Namespace, ch.Metadata.Name, key.namespace, key.name)
	}
	ns.Endpoint = append(ns.Endpoint, ch)

	return nil
}

// DeleteEndpoint deletes Endpoint from Network Service
func (n *networkServicesStore) DeleteEndpointFromNetworkService(nsName string, nsNamespace string, ch *netmesh.NetworkServiceEndpoint) error {
	n.Lock()
	defer n.Unlock()

	key := meta{
		name:      nsName,
		namespace: nsNamespace,
	}
	ns, ok := n.networkService[key]
	if !ok {
		return fmt.Errorf("failed to find network service %s/%s in the object store", key.namespace, key.name)
	}

	// Need to check if NetworkService has already Endpoint with the same name and namespace, Endpoints must
	// be unique for Name and Namspace pair.
	for i, c := range ns.Endpoint {
		if c.Metadata.Name == ch.Metadata.Name && c.Metadata.Namespace == ch.Metadata.Namespace {
			ns.Endpoint = append(ns.Endpoint[:i], ns.Endpoint[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("failed to delete Endpoint %s/%s from network service %s/%s, the Endpoint does not exist in the object store",
		ch.Metadata.Namespace, ch.Metadata.Name, key.namespace, key.name)
}
*/
// Delete method deletes removed NetworkService object from the store.
func (n *networkServicesStore) Delete(ns string) {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.networkService[ns]; ok {
		delete(n.networkService, ns)
	}
}

// List method lists all known NetworkService objects.
func (n *networkServicesStore) List() []*v1.NetworkService {
	n.Lock()
	defer n.Unlock()
	networkServices := make([]*v1.NetworkService, 0)
	for _, ns := range n.networkService {
		networkServices = append(networkServices, ns)
	}
	return networkServices
}

/*
func (n *networkServicesStore) ListEndpointsForNetworkService(ns *netmesh.NetworkService) ([]*netmesh.NetworkServiceEndpoint, error) {
	n.Lock()
	defer n.Unlock()

	key := meta{
		name:      ns.Metadata.Name,
		namespace: ns.Metadata.Namespace,
	}
	ns, ok := n.networkService[key]
	if !ok {
		return nil, fmt.Errorf("failed to find network service %s/%s in the object store", key.namespace, key.name)
	}
	return ns.Endpoint, nil
}
*/
