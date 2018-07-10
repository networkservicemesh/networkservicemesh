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

type endpointStore struct {
	*networkServiceEndpointsStore
}

const (
	epTestName      = "EndpointTest"
	epTestNamespace = "default"
)

func TestEndpointStore(t *testing.T) {
	endpoints := &endpointStore{}
	endpoints.networkServiceEndpointsStore = newNetworkServiceEndpointsStore()

	nse := netmesh.NetworkServiceEndpoint{
		Metadata: &common.Metadata{
			Name:      epTestName,
			Namespace: epTestNamespace,
		},
	}

	endpoints.networkServiceEndpointsStore.Add(&nse)

	for _, n := range endpoints.networkServiceEndpointsStore.List() {
		if n.Metadata.Name != epTestName {
			t.Errorf("Unexpected Name value returned when creating NetworkServiceEndpoint")
		}

		if n.Metadata.Namespace != epTestNamespace {
			t.Errorf("Unexpected Namespace value returned when creating NetworkServiceEndpoint")
		}
	}

	endpoints.networkServiceEndpointsStore.Delete(meta{name: nse.Metadata.Name, namespace: nse.Metadata.Namespace})

	nseRet := endpoints.networkServiceEndpointsStore.List()
	if len(nseRet) != 0 {
		t.Errorf("Deletion of NetworkServiceEndpoint from objectstore failed")
	}
}
