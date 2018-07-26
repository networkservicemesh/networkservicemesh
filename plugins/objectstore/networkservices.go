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
	"fmt"
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
func (n *networkServicesStore) Get(nsName, nsNamespace string) *netmesh.NetworkService {
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

// Get method returns NetworkService, if it does not
// already it returns nil.
func (n *networkServicesStore) AddChannelToNetworkService(nsName string, nsNamespace string, ch *netmesh.NetworkServiceChannel) error {
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

	// Need to check if NetworkService has already Channel with the same name and namespace, Channels must
	// be unique for Name and Namspace pair.
	found := false
	for _, c := range ns.Channel {
		if c.Metadata.Name == ch.Metadata.Name && c.Metadata.Namespace == ch.Metadata.Namespace {
			found = true
			break
		}
	}
	if found {
		return fmt.Errorf("failed to add channel %s/%s to network service %s/%s, the channel already exists in the object store",
			ch.Metadata.Namespace, ch.Metadata.Name, key.namespace, key.name)
	}
	ns.Channel = append(ns.Channel, ch)

	return nil
}

// DeleteChannel deletes channel from Network Service
func (n *networkServicesStore) DeleteChannelFromNetworkService(nsName string, nsNamespace string, ch *netmesh.NetworkServiceChannel) error {
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

	// Need to check if NetworkService has already Channel with the same name and namespace, Channels must
	// be unique for Name and Namspace pair.
	for i, c := range ns.Channel {
		if c.Metadata.Name == ch.Metadata.Name && c.Metadata.Namespace == ch.Metadata.Namespace {
			ns.Channel = append(ns.Channel[:i], ns.Channel[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("failed to delete channel %s/%s from network service %s/%s, the channel does not exist in the object store",
		ch.Metadata.Namespace, ch.Metadata.Name, key.namespace, key.name)
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

func (n *networkServicesStore) ListChannelsForNetworkService(ns *netmesh.NetworkService) ([]*netmesh.NetworkServiceChannel, error) {
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
	return ns.Channel, nil
}
