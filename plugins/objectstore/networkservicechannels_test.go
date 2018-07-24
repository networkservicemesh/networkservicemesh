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
	"testing"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
)

type channelStore struct {
	*networkServiceChannelsStore
}

func TestChannelStore(t *testing.T) {
	channels := &channelStore{}
	channels.networkServiceChannelsStore = newNetworkServiceChannelsStore()

	nsc1 := netmesh.NetworkServiceChannel{
		Metadata: &common.Metadata{
			Name:      "channel-1",
			Namespace: "default",
		},
		NseProviderName:    "host1",
		NetworkServiceName: "network-service-1",
	}

	nsc2 := netmesh.NetworkServiceChannel{
		Metadata: &common.Metadata{
			Name:      "channel-2",
			Namespace: "default",
		},
		NseProviderName:    "host1",
		NetworkServiceName: "network-service-1",
	}

	channels.networkServiceChannelsStore.AddChannel(&nsc1)
	channels.networkServiceChannelsStore.AddChannel(&nsc2)

	nseChannels := channels.networkServiceChannelsStore.GetChannelsByNSEServerProvider("host1", "default")
	if len(nseChannels) != 2 {
		t.Fatalf("expected to get exactly 2 channels but got %d", len(nseChannels))
	}
	channels.networkServiceChannelsStore.DeleteChannel(&nsc1)
	nseChannels = channels.networkServiceChannelsStore.GetChannelsByNSEServerProvider("host1", "default")
	if len(nseChannels) != 1 {
		t.Fatalf("expected to get exactly 1 channels but got %d", len(nseChannels))
	}
	channels.networkServiceChannelsStore.DeleteChannel(&nsc2)
	nseChannels = channels.networkServiceChannelsStore.GetChannelsByNSEServerProvider("host1", "default")
	if len(nseChannels) != 0 {
		t.Fatalf("expected to get exactly 0 channels but got %d", len(nseChannels))
	}
}
