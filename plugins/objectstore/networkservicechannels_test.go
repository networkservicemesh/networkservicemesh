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

	"github.com/ligato/networkservicemesh/netmesh/model/netmesh"
)

type channelStore struct {
	*networkServiceChannelsStore
}

const (
	chTestName      = "ChannelTest"
	chTestNamespace = "default"
)

var channels *channelStore

func TestChannelStore(t *testing.T) {
	channels := &channelStore{}
	channels.networkServiceChannelsStore = newNetworkServiceChannelsStore()

	nsc := netmesh.NetworkService_NetmeshChannel{
		Metadata: &netmesh.Metadata{},
	}

	nsc = netmesh.NetworkService_NetmeshChannel{
		Metadata: &netmesh.Metadata{
			Name:      chTestName,
			Namespace: chTestNamespace,
		},
	}

	channels.networkServiceChannelsStore.Add(&nsc)

	for _, n := range channels.networkServiceChannelsStore.List() {
		if n.Metadata.Name != chTestName {
			t.Errorf("Unexpected Name value returned when creating NetworkServiceChannel")
		}

		if n.Metadata.Namespace != chTestNamespace {
			t.Errorf("Unexpected Namespace value returned when creating NetworkServiceChannel")
		}
	}

	channels.networkServiceChannelsStore.Delete(meta{name: nsc.Metadata.Name, namespace: nsc.Metadata.Namespace})

	nscRet := channels.networkServiceChannelsStore.List()
	if len(nscRet) != 0 {
		t.Errorf("Deletion of NetworkServiceChannel from objectstore failed")
	}
}
