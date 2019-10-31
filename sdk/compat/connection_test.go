// Copyright (c) 2019 Cisco Systems, Inc.
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

package compat_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	local_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/sdk/compat"
)

func TestConnectionEventUnifiedToRemoteToUnified1(t *testing.T) {
	g := NewWithT(t)
	unifiedConnectionEvent := &connection.ConnectionEvent{
		Type: connection.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: map[string]*connection.Connection{
			"1": &connection.Connection{
				Id:             "1",
				NetworkService: "golden_network",
				Mechanism: &connection.Mechanism{
					Cls:        cls.REMOTE,
					Type:       vxlan.MECHANISM,
					Parameters: nil,
				},
				Context: &connectioncontext.ConnectionContext{
					IpContext: &connectioncontext.IPContext{
						SrcIpAddr:          "",
						DstIpAddr:          "",
						SrcIpRequired:      true,
						DstIpRequired:      true,
						SrcRoutes:          nil,
						DstRoutes:          nil,
						ExcludedPrefixes:   nil,
						IpNeighbors:        nil,
						ExtraPrefixRequest: nil,
						ExtraPrefixes:      nil,
					},
					DnsContext: &connectioncontext.DNSContext{
						Configs: nil,
					},
				},
			},
		},
	}
	testConnectionEventUnifiedToRemoteToUnified(t, g, unifiedConnectionEvent, nil)
}

func testConnectionEventRemoteToUnifiedToRemote(t *testing.T, g *WithT, unifiedConnectionEvents ...*remote_connection.ConnectionEvent) {
	for _, unifiedConnectionEvent := range unifiedConnectionEvents {
		remoteRequest := compat.ConnectionEventRemoteToUnified(unifiedConnectionEvent)
		roundTripRequest := compat.ConnectionEventUnifiedToRemote(remoteRequest)
		g.Expect(roundTripRequest).To(Equal(unifiedConnectionEvent))
	}
}

func testConnectionEventUnifiedToRemoteToUnified(t *testing.T, g *WithT, unifiedConnectionEvents ...*connection.ConnectionEvent) {
	for _, unifiedConnectionEvent := range unifiedConnectionEvents {
		remoteRequest := compat.ConnectionEventUnifiedToRemote(unifiedConnectionEvent)
		roundTripRequest := compat.ConnectionEventRemoteToUnified(remoteRequest)
		g.Expect(roundTripRequest).To(Equal(unifiedConnectionEvent))
	}
}

func TestConnectionEventUnifiedToLocalToUnified1(t *testing.T) {
	g := NewWithT(t)
	unifiedConnectionEvent := &connection.ConnectionEvent{
		Type: connection.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: map[string]*connection.Connection{
			"1": &connection.Connection{
				Id:             "1",
				NetworkService: "golden_network",
				Mechanism: &connection.Mechanism{
					Cls:        cls.LOCAL,
					Type:       memif.MECHANISM,
					Parameters: nil,
				},
				Context: &connectioncontext.ConnectionContext{
					IpContext: &connectioncontext.IPContext{
						SrcIpAddr:          "",
						DstIpAddr:          "",
						SrcIpRequired:      true,
						DstIpRequired:      true,
						SrcRoutes:          nil,
						DstRoutes:          nil,
						ExcludedPrefixes:   nil,
						IpNeighbors:        nil,
						ExtraPrefixRequest: nil,
						ExtraPrefixes:      nil,
					},
					DnsContext: &connectioncontext.DNSContext{
						Configs: nil,
					},
				},
				NetworkServiceManagers: make([]string, 1),
			},
		},
	}
	testConnectionEventUnifiedToLocalToUnified(t, g, unifiedConnectionEvent, nil)
}

func testConnectionEventLocalToUnifiedToLocal(t *testing.T, g *WithT, unifiedConnectionEvents ...*local_connection.ConnectionEvent) {
	for _, unifiedConnectionEvent := range unifiedConnectionEvents {
		localRequest := compat.ConnectionEventLocalToUnified(unifiedConnectionEvent)
		roundTripRequest := compat.ConnectionEventUnifiedToLocal(localRequest)
		g.Expect(roundTripRequest).To(Equal(unifiedConnectionEvent))
	}
}

func testConnectionEventUnifiedToLocalToUnified(t *testing.T, g *WithT, unifiedConnectionEvents ...*connection.ConnectionEvent) {
	for _, unifiedConnectionEvent := range unifiedConnectionEvents {
		localRequest := compat.ConnectionEventUnifiedToLocal(unifiedConnectionEvent)
		roundTripRequest := compat.ConnectionEventLocalToUnified(localRequest)
		g.Expect(roundTripRequest).To(Equal(unifiedConnectionEvent))
	}
}

func TestMonitorScopeSelectorUnifiedToRemoteToUnified1(t *testing.T) {
	g := NewWithT(t)
	unifiedMonitorScopeSelector := &connection.MonitorScopeSelector{
		NetworkServiceManagers: []string{
			"nsmgr1",
			"nsmgr2",
		},
	}

	testMonitorScopeSelectorUnifiedToRemoteToUnified(t, g, unifiedMonitorScopeSelector, nil)

}

func testMonitorScopeSelectorUnifiedToRemoteToUnified(t *testing.T, g *WithT, unifiedMonitorScopeSelectors ...*connection.MonitorScopeSelector) {
	for _, unifiedMonitorScopeSelector := range unifiedMonitorScopeSelectors {
		remoteMonitorScopeSelector := compat.MonitorScopeSelectorUnifiedToRemote(unifiedMonitorScopeSelector)
		roundTripMonitorScopeSelector := compat.MonitorScopeSelectorRemoteToUnified(remoteMonitorScopeSelector)
		g.Expect(roundTripMonitorScopeSelector).To(Equal(unifiedMonitorScopeSelector))
	}
}

func TestMonitorScopeSelectorRemoteToUnifiedToRemote1(t *testing.T) {
	g := NewWithT(t)
	remoteMonitorScopeSelector := &remote_connection.MonitorScopeSelector{}
	remoteMonitorScopeSelector2 := &remote_connection.MonitorScopeSelector{
		NetworkServiceManagerName:            "nsm1",
		DestinationNetworkServiceManagerName: "",
	}
	remoteMonitorScopeSelector3 := &remote_connection.MonitorScopeSelector{
		NetworkServiceManagerName:            "nsm1",
		DestinationNetworkServiceManagerName: "nsm2",
	}
	remoteMonitorScopeSelector4 := &remote_connection.MonitorScopeSelector{
		NetworkServiceManagerName:            "",
		DestinationNetworkServiceManagerName: "nsm2",
	}
	testMonitorScopeSelectorRemoteToUnifiedToRemote(t, g, remoteMonitorScopeSelector, remoteMonitorScopeSelector2, remoteMonitorScopeSelector3, remoteMonitorScopeSelector4, nil)
}

func testMonitorScopeSelectorRemoteToUnifiedToRemote(t *testing.T, g *WithT, remoteMonitorScopeSelectors ...*remote_connection.MonitorScopeSelector) {
	for _, remoteMonitorScopeSelector := range remoteMonitorScopeSelectors {
		unifiedMonitorScopeSelector := compat.MonitorScopeSelectorRemoteToUnified(remoteMonitorScopeSelector)
		roundTripMonitorScopeSelector := compat.MonitorScopeSelectorUnifiedToRemote(unifiedMonitorScopeSelector)
		g.Expect(roundTripMonitorScopeSelector).To(Equal(remoteMonitorScopeSelector))
	}
}
