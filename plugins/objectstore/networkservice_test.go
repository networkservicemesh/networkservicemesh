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

type networkserviceStore struct {
	*networkServicesStore
}

const (
	nsTestName      = "EndpointTest"
	nsTestNamespace = "default"
)

func TestNetworkServiceStore(t *testing.T) {
	networkservices := &networkserviceStore{}
	networkservices.networkServicesStore = newNetworkServicesStore()

	ns := netmesh.NetworkService{
		Metadata: &common.Metadata{
			Name:      nsTestName,
			Namespace: nsTestNamespace,
		},
	}

	networkservices.networkServicesStore.Add(&ns)

	for _, n := range networkservices.networkServicesStore.List() {
		if n.Metadata.Name != nsTestName {
			t.Errorf("Unexpected Name value returned when creating NetworkService")
		}

		if n.Metadata.Namespace != nsTestNamespace {
			t.Errorf("Unexpected Namespace value returned when creating NetworkService")
		}
	}

	networkservices.networkServicesStore.Delete(meta{name: ns.Metadata.Name, namespace: ns.Metadata.Namespace})

	nsRet := networkservices.networkServicesStore.List()
	if len(nsRet) != 0 {
		t.Errorf("Deletion of NetworkService from objectstore failed")
	}
}

// TODO (sbezverk) AddChannelToNetworkService and DeleteChannelFromNetworkService
