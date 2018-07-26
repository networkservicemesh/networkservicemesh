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
	nsTestName          = "EndpointTest"
	nsTestNamespace     = "default"
	nchTestName         = "channel-1"
	nchTestNamespace    = "default"
	nseTestProviderName = "nse-1"
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

func TestAddChannelToNetworkService(t *testing.T) {
	networkservices := &networkserviceStore{}
	networkservices.networkServicesStore = newNetworkServicesStore()

	ns := netmesh.NetworkService{
		Metadata: &common.Metadata{
			Name:      nsTestName,
			Namespace: nsTestNamespace,
		},
	}
	nch := netmesh.NetworkServiceChannel{
		Metadata: &common.Metadata{
			Name:      nchTestName,
			Namespace: nchTestNamespace,
		},
		NetworkServiceName: nsTestName,
		NseProviderName:    nseTestProviderName,
	}

	networkservices.networkServicesStore.Add(&ns)

	if err := networkservices.networkServicesStore.AddChannelToNetworkService(nsTestName, nsTestNamespace, &nch); err != nil {
		t.Fatalf("AddChannel failed with error: %+v", err)
	}
	resultChannels, err := networkservices.networkServicesStore.ListChannelsForNetworkService(&ns)
	if err != nil {
		t.Fatalf("List channels for network service %s/%s failed with error: %+v", ns.Metadata.Namespace, ns.Metadata.Name, err)
	}
	if resultChannels == nil {
		t.Fatalf("Channel slice for network service %s/%s should not be nil, but it is, failing", ns.Metadata.Namespace, ns.Metadata.Name)
	}
	if len(resultChannels) != 1 {
		t.Fatalf("Expected to see exactly 1 channel but got %d, failing", len(resultChannels))
	}
}

func TestDeletehannelToNetworkService(t *testing.T) {

	networkservices := &networkserviceStore{}
	networkservices.networkServicesStore = newNetworkServicesStore()

	ns := netmesh.NetworkService{
		Metadata: &common.Metadata{
			Name:      nsTestName,
			Namespace: nsTestNamespace,
		},
	}
	nch := netmesh.NetworkServiceChannel{
		Metadata: &common.Metadata{
			Name:      nchTestName,
			Namespace: nchTestNamespace,
		},
		NetworkServiceName: nsTestName,
		NseProviderName:    nseTestProviderName,
	}

	networkservices.networkServicesStore.Add(&ns)

	if err := networkservices.networkServicesStore.AddChannelToNetworkService(nsTestName, nsTestNamespace, &nch); err != nil {
		t.Fatalf("AddChannelToNetworkService failed with error: %+v", err)
	}
	resultChannels, err := networkservices.networkServicesStore.ListChannelsForNetworkService(&ns)
	if err != nil {
		t.Fatalf("List channels for network service %s/%s failed with error: %+v", ns.Metadata.Namespace, ns.Metadata.Name, err)
	}
	if resultChannels == nil {
		t.Fatalf("Channel slice for network service %s/%s should not be nil, but it is, failing", ns.Metadata.Namespace, ns.Metadata.Name)
	}
	if len(resultChannels) != 1 {
		t.Fatalf("Expected to see exactly 1 channel but got %d, failing", len(resultChannels))
	}
	if err := networkservices.networkServicesStore.DeleteChannelFromNetworkService(nsTestName, nsTestNamespace, &nch); err != nil {
		t.Fatalf("DeleteChannelToNetworkService failed with error: %+v", err)
	}
	resultChannels, err = networkservices.networkServicesStore.ListChannelsForNetworkService(&ns)
	if err != nil {
		t.Fatalf("List channels for network service %s/%s failed with error: %+v", ns.Metadata.Namespace, ns.Metadata.Name, err)
	}
	if len(resultChannels) != 0 {
		t.Fatalf("Expected to see exactly 0 channel but got %d, failing", len(resultChannels))
	}
}
