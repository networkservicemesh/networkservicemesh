// Copyright 2018 VMware, Inc.
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

package selector

import (
	"sync"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
)

type matchSelector struct {
	sync.Mutex
	roundRobin Selector
}

// NewMatchSelector creates a new
func NewMatchSelector() Selector {
	return &matchSelector{
		roundRobin: NewRoundRobinSelector(),
	}
}

// isSubset checks if B is a subset of A. TODO: reconsider this as a part of "tools"
func isSubset(A, B map[string]string) bool {
	if len(A) < len(B) {
		return false
	}
	for k, v := range B {
		if A[k] != v {
			return false
		}
	}
	return true
}

func (m *matchSelector) matchEndpoint(nsLabels map[string]string, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	//Iterate through the matches
	for _, match := range ns.GetMatches() {
		// All match source selector labels should be present in the requested labels map
		if !isSubset(nsLabels, match.GetSourceSelector()) {
			break
		}

		nseCandidates := []*registry.NetworkServiceEndpoint{}
		// Check all Destinations in that match
		for _, destination := range match.GetRoutes() {
			// Each NSE should be matched against that destination
			for _, nse := range networkServiceEndpoints {
				if isSubset(nse.GetLabels(), destination.GetDestinationSelector()) {
					nseCandidates = append(nseCandidates, nse)
				}
			}
		}

		if len(nseCandidates) > 0 {
			// We found candidates. Use RoundRobin to select one
			return m.roundRobin.SelectEndpoint(nil, ns, nseCandidates)
		}
	}
	return nil
}

func (m *matchSelector) SelectEndpoint(requestConnection *connection.Connection, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	if len(ns.GetMatches()) == 0 {
		return m.roundRobin.SelectEndpoint(nil, ns, networkServiceEndpoints)
	}

	return m.matchEndpoint(requestConnection.GetLabels(), ns, networkServiceEndpoints)
}
