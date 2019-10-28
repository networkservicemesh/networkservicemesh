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
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/compat"
)

func TestNetworkServiceUnifiedToRemoteToUnified1(t *testing.T) {
	g := NewWithT(t)
	unifiedRequest := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
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
			Labels:                     nil,
			NetworkServiceManagers:     nil,
			NetworkServiceEndpointName: "ep1",
			State:                      connection.State_UP,
		},
		MechanismPreferences: nil,
	}
	unifiedRequest2 := unifiedRequest.Clone()
	unifiedRequest2.MechanismPreferences = []*connection.Mechanism{
		&connection.Mechanism{
			Cls:  cls.REMOTE,
			Type: vxlan.MECHANISM,
			Parameters: map[string]string{
				"src_ip": "127.0.0.1",
			},
		},
	}

	unifiedRequest3 := unifiedRequest.Clone()
	unifiedRequest3.Connection = nil

	unifiedRequest4 := unifiedRequest.Clone()
	unifiedRequest4.GetConnection().Mechanism = nil
	testUnifiedToRemoteToUnified(t, g, unifiedRequest, unifiedRequest2, unifiedRequest3, unifiedRequest4, nil)
}
func testUnifiedToRemoteToUnified(t *testing.T, g *WithT, unifiedRequests ...*networkservice.NetworkServiceRequest) {
	for _, unifiedRequest := range unifiedRequests {
		remoteRequest := compat.NetworkServiceRequestUnifiedToRemote(unifiedRequest)
		roundTripRequest := compat.NetworkServiceRequestRemoteToUnified(remoteRequest)
		g.Expect(roundTripRequest).To(Equal(unifiedRequest))
	}
}

func TestNetworkServiceRemoteToUnifiedToRemote1(t *testing.T) {
	g := NewWithT(t)
	remoteRequest := &remote.NetworkServiceRequest{
		Connection: &remote_connection.Connection{
			Id:             "1",
			NetworkService: "golden_network",
			Mechanism: &remote_connection.Mechanism{
				Type:       remote_connection.MechanismType_VXLAN,
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
			Labels:                               nil,
			SourceNetworkServiceManagerName:      "",
			DestinationNetworkServiceManagerName: "",
			NetworkServiceEndpointName:           "",
			State:                                0,
		},
		MechanismPreferences: nil,
	}

	remoteRequest2 := remoteRequest.Clone().(*remote.NetworkServiceRequest)
	remoteRequest2.MechanismPreferences = []*remote_connection.Mechanism{
		&remote_connection.Mechanism{
			Type: remote_connection.MechanismType_VXLAN,
			Parameters: map[string]string{
				vxlan.SrcIP: "127.0.0.1",
			},
		},
	}

	remoteRequest3 := remoteRequest.Clone().(*remote.NetworkServiceRequest)
	remoteRequest3.Connection = nil

	remoteRequest4 := remoteRequest.Clone().(*remote.NetworkServiceRequest)
	remoteRequest4.GetConnection().Mechanism = nil

	testRemoteToUnifiedToRemote(t, g, remoteRequest, remoteRequest2, remoteRequest3, remoteRequest4, nil)
}
func testRemoteToUnifiedToRemote(t *testing.T, g *WithT, remoteRequests ...*remote.NetworkServiceRequest) {
	for _, remoteRequest := range remoteRequests {
		unifiedRequest := compat.NetworkServiceRequestRemoteToUnified(remoteRequest)
		roundTripRequest := compat.NetworkServiceRequestUnifiedToRemote(unifiedRequest)
		g.Expect(roundTripRequest).To(Equal(remoteRequest))
	}
}

func TestNetworkServiceUnifiedToLocalToUnified1(t *testing.T) {
	g := NewWithT(t)
	unifiedRequest := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
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
			Labels:                     nil,
			NetworkServiceManagers:     nil,
			NetworkServiceEndpointName: "ep1",
			State:                      connection.State_UP,
		},
		MechanismPreferences: nil,
	}
	unifiedRequest2 := unifiedRequest.Clone()
	unifiedRequest2.MechanismPreferences = []*connection.Mechanism{
		&connection.Mechanism{
			Cls:  cls.LOCAL,
			Type: memif.MECHANISM,
			Parameters: map[string]string{
				memif.SocketFilename: memif.MemifSocket,
			},
		},
	}

	unifiedRequest3 := unifiedRequest.Clone()
	unifiedRequest3.Connection = nil

	unifiedRequest4 := unifiedRequest.Clone()
	unifiedRequest4.GetConnection().Mechanism = nil

	testUnifiedToLocalToUnified(t, g, unifiedRequest, unifiedRequest2, unifiedRequest3, unifiedRequest4, nil)
}

func testUnifiedToLocalToUnified(t *testing.T, g *WithT, unifiedRequests ...*networkservice.NetworkServiceRequest) {
	for _, unifiedRequest := range unifiedRequests {
		localRequest := compat.NetworkServiceRequestUnifiedToLocal(unifiedRequest)
		roundTripRequest := compat.NetworkServiceRequestLocalToUnified(localRequest)
		// localRequest has no NetworkService Endpoint or Network Service Managers... so we fudge it here
		if roundTripRequest.GetConnection() != nil && unifiedRequest.GetConnection() != nil {
			roundTripRequest.GetConnection().NetworkServiceManagers = unifiedRequest.GetConnection().GetNetworkServiceManagers()
			roundTripRequest.GetConnection().NetworkServiceEndpointName = unifiedRequest.GetConnection().GetNetworkServiceEndpointName()
		}
		g.Expect(roundTripRequest).To(Equal(unifiedRequest))
	}
}

func TestNetworkServiceLocalToUnifiedToLocal1(t *testing.T) {
	g := NewWithT(t)
	localRequest := &local.NetworkServiceRequest{
		Connection: &local_connection.Connection{
			Id:             "1",
			NetworkService: "golden_network",
			Mechanism: &local_connection.Mechanism{
				Type:       local_connection.MechanismType_MEM_INTERFACE,
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
			Labels: nil,
			State:  0,
		},
		MechanismPreferences: nil,
	}

	localRequest2 := localRequest.Clone().(*local.NetworkServiceRequest)
	localRequest2.MechanismPreferences = []*local_connection.Mechanism{
		&local_connection.Mechanism{
			Type: local_connection.MechanismType_MEM_INTERFACE,
			Parameters: map[string]string{
				memif.SocketFilename: memif.MemifSocket,
			},
		},
	}

	localRequest3 := localRequest.Clone().(*local.NetworkServiceRequest)
	localRequest3.Connection = nil

	localRequest4 := localRequest.Clone().(*local.NetworkServiceRequest)
	localRequest4.GetConnection().Mechanism = nil

	testLocalToUnifiedToLocal(t, g, localRequest, localRequest2, localRequest3, localRequest4, nil)
}

func testLocalToUnifiedToLocal(t *testing.T, g *WithT, remoteRequests ...*local.NetworkServiceRequest) {
	for _, localRequest := range remoteRequests {
		unifiedRequest := compat.NetworkServiceRequestLocalToUnified(localRequest)
		roundTripRequest := compat.NetworkServiceRequestUnifiedToLocal(unifiedRequest)
		g.Expect(roundTripRequest).To(Equal(localRequest))
	}
}
