// Copyright (c) 2019 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
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

package tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/applications/nsmrs/pkg/serviceregistryserver"
)

func TestNSMRSCacheAdd(t *testing.T) {
	g := NewWithT(t)

	cache := serviceregistryserver.NewNSERegistryCache()

	nse := newTestNse("nse1", "ns1")

	_, err := cache.AddNetworkServiceEndpoint(nse)
	g.Expect(err).To(BeNil())

	endpointList := cache.GetEndpoints("ns1")
	g.Expect(len(endpointList)).To(Equal(1))
	g.Expect(endpointList[0].NetworkServiceEndpoint.Name).To(Equal("nse1"))
}

func TestNSMRSCacheDelete(t *testing.T) {
	g := NewWithT(t)

	cache := serviceregistryserver.NewNSERegistryCache()

	nse := newTestNse("nse1", "ns1")

	_, err := cache.AddNetworkServiceEndpoint(nse)
	g.Expect(err).To(BeNil())
	endpointList := cache.GetEndpoints("ns1")
	g.Expect(len(endpointList)).To(Equal(1))

	endpoint, err := cache.DeleteNetworkServiceEndpoint("nse1")
	g.Expect(err).To(BeNil())
	g.Expect(endpoint.NetworkServiceEndpoint.Name).To(Equal("nse1"))

	endpointList = cache.GetEndpoints("ns1")
	g.Expect(len(endpointList)).To(Equal(0))
}

func TestNSMRSCacheNSECollision(t *testing.T) {
	g := NewWithT(t)

	cache := serviceregistryserver.NewNSERegistryCache()
	nse1 := newTestNse("nse1", "ns1")
	_, err := cache.AddNetworkServiceEndpoint(nse1)
	g.Expect(err).To(BeNil())

	nse2 := newTestNse("nse2", "ns2")
	_, err = cache.AddNetworkServiceEndpoint(nse2)
	g.Expect(err).To(BeNil())

	nse1clone := newTestNse("nse1", "ns1")
	_, err = cache.AddNetworkServiceEndpoint(nse1clone)
	g.Expect(err.Error()).To(ContainSubstring("already exists"))
}

func TestNSMRSCacheNSCollision(t *testing.T) {
	g := NewWithT(t)

	cache := serviceregistryserver.NewNSERegistryCache()
	nse1 := newTestNseWithPayload("nse1", "ns", "IP")
	_, err := cache.AddNetworkServiceEndpoint(nse1)
	g.Expect(err).To(BeNil())

	nse2 := newTestNseWithPayload("nse2", "ns", "IP")
	_, err = cache.AddNetworkServiceEndpoint(nse2)
	g.Expect(err).To(BeNil())

	nse1clone := newTestNseWithPayload("nse3", "ns", "TCP")
	_, err = cache.AddNetworkServiceEndpoint(nse1clone)
	g.Expect(err.Error()).To(ContainSubstring("network service already exists with different parameters"))
}
