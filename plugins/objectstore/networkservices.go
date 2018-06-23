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

	"github.com/ligato/networkservicemesh/netmesh/model/netmesh"
)

// NetworkServicesStore map stores all discovered Network Service Object
// with a key composed of a name and a namespace
type NetworkServicesStore struct {
	NetworkService map[meta]*netmesh.NetworkService
	sync.RWMutex
}

// NewNetworkServicesStore instantiates a new instance of a global
// NetworkServices store. It must be initialized before any controllers start.
func newNetworkServicesStore() *NetworkServicesStore {
	return &NetworkServicesStore{
		NetworkService: map[meta]*netmesh.NetworkService{}}
}

// Add method adds descovered NetworkService if it does not
// already exit in the store.
func (n *NetworkServicesStore) Add(ns *netmesh.NetworkService) {
	n.Lock()
	defer n.Unlock()

	key := meta{
		name: ns.Name,
		// TODO replace it with namespace
		namespace: ns.Uuid,
	}
	if _, ok := n.NetworkService[key]; !ok {
		// Not in the store, adding it.
		n.NetworkService[key] = ns
	}
}

// Delete method deletes removed NetworkService object from the store.
func (n *NetworkServicesStore) Delete(key meta) {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.NetworkService[key]; ok {
		delete(n.NetworkService, key)
	}
}

// List method lists all known NetworkService objects.
func (n *NetworkServicesStore) List() []*netmesh.NetworkService {
	n.Lock()
	defer n.Unlock()
	networkServices := []*netmesh.NetworkService{}
	for _, ns := range n.NetworkService {
		networkServices = append(networkServices, ns)
	}

	return networkServices
}
