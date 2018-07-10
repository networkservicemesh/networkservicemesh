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

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
)

// NetworkServicesStore map stores all discovered Network Service Object
// with a key composed of a name and a namespace
type networkServicesStore struct {
	networkService map[meta]*netmesh.NetworkService
	sync.RWMutex
}

// NewNetworkServicesStore instantiates a new instance of a global
// NetworkServices store. It must be initialized before any controllers start.
func newNetworkServicesStore() *networkServicesStore {
	return &networkServicesStore{
		networkService: map[meta]*netmesh.NetworkService{}}
}

// Add method adds descovered NetworkService if it does not
// already exit in the store.
func (n *networkServicesStore) Add(ns *netmesh.NetworkService) {
	n.Lock()
	defer n.Unlock()

	key := meta{
		name:      ns.Metadata.Name,
		namespace: ns.Metadata.Namespace,
	}
	if _, ok := n.networkService[key]; !ok {
		// Not in the store, adding it.
		n.networkService[key] = ns
	}
}

// Get method returns NetworkService, if it does not
// already it returns nil.
func (n *networkServicesStore) Get(nsName string, nsNamespace string) *netmesh.NetworkService {
	n.Lock()
	defer n.Unlock()

	key := meta{
		name:      nsName,
		namespace: nsNamespace,
	}
	ns, ok := n.networkService[key]
	if !ok {
		return nil
	}
	return ns
}

// Delete method deletes removed NetworkService object from the store.
func (n *networkServicesStore) Delete(key meta) {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.networkService[key]; ok {
		delete(n.networkService, key)
	}
}

// List method lists all known NetworkService objects.
func (n *networkServicesStore) List() []*netmesh.NetworkService {
	n.Lock()
	defer n.Unlock()
	networkServices := make([]*netmesh.NetworkService, 0)
	for _, ns := range n.networkService {
		networkServices = append(networkServices, ns)
	}
	return networkServices
}
