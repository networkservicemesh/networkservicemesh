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

// NetworkServiceChannelsStore map stores all discovered Network Service Channel
// Objects with a key composed of a name and a namespace
type networkServiceChannelsStore struct {
	networkServiceChannel map[meta][]*netmesh.NetworkServiceChannel
	sync.RWMutex
}

// NewNetworkServiceChannelsStore instantiates a new instance of a global
// NetworkServiceChannels store. It must be initialized before any controllers start.
func newNetworkServiceChannelsStore() *networkServiceChannelsStore {
	return &networkServiceChannelsStore{
		networkServiceChannel: map[meta][]*netmesh.NetworkServiceChannel{}}
}

// Add method adds descovered NetworkServiceChannel if it does not
// already exit in the store.
func (n *networkServiceChannelsStore) AddChannel(nch *netmesh.NetworkServiceChannel) {
	n.Lock()
	defer n.Unlock()

	key := meta{
		name:      nch.NseProviderName,
		namespace: nch.Metadata.Namespace,
	}
	found := false
	if channels, ok := n.networkServiceChannel[key]; !ok {
		// Not in the store, meaning it will be a first channel for NSE
		n.networkServiceChannel[key] = append(n.networkServiceChannel[key], nch)
	} else {
		// NSE already exists, now need to check is the channel is not duplicate
		// and if it is not, then add it.
		for _, ch := range channels {
			if ch.Metadata.Name == nch.Metadata.Name &&
				ch.Metadata.Namespace == nch.Metadata.Namespace {
				found = true
			}
		}
		if !found {
			n.networkServiceChannel[key] = append(n.networkServiceChannel[key], nch)
		}
	}
}

// Delete method deletes removed NetworkServiceChannel object from the store.
func (n *networkServiceChannelsStore) DeleteChannel(nch *netmesh.NetworkServiceChannel) {
	n.Lock()
	defer n.Unlock()

	key := meta{
		name:      nch.NseProviderName,
		namespace: nch.Metadata.Namespace,
	}
	if channels, ok := n.networkServiceChannel[key]; ok {
		for i, c := range channels {
			if c.Metadata.Name == nch.Metadata.Name && c.Metadata.Namespace == nch.Metadata.Namespace {
				n.networkServiceChannel[key] = append(n.networkServiceChannel[key][:i], n.networkServiceChannel[key][i+1:]...)
			}
		}
	}
}

// DeleteNSE method deletes removed NetworkServiceChannel object from the store.
func (n *networkServiceChannelsStore) DeleteNSE(nseServer, namespace string) {
	n.Lock()
	defer n.Unlock()

	key := meta{
		name:      nseServer,
		namespace: namespace,
	}
	if _, ok := n.networkServiceChannel[key]; ok {
		delete(n.networkServiceChannel, key)
	}
}

// GetChannelsByNSEServerProvider returns a slice of channels for specified nse_server_provider + namespace key
func (n *networkServiceChannelsStore) GetChannelsByNSEServerProvider(nseServer, namespace string) []*netmesh.NetworkServiceChannel {
	key := meta{
		name:      nseServer,
		namespace: namespace,
	}
	if list, ok := n.networkServiceChannel[key]; ok {
		return list
	}
	return nil
}
