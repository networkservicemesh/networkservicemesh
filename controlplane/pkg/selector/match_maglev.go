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

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type MatchMaglevSelector struct {
	sync.Mutex
	maglev Selector
}

// NewMaglevSelector creates a new
func NewMatchMaglevSelector() Selector {
	//logrus.Info("start maglev selector ")
	return &MatchMaglevSelector{
		maglev: NewmaglevSelector(),
	}
}



// isSubset checks if B is a subset of A. TODO: reconsider this as a part of "tools"
func ismaglevSubset(A, B map[string]string) bool {
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

func (mg *MatchMaglevSelector) maglevEndpoint(requestConnection *connection.Connection, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	logrus.Infof("****************** Maglev ednpoint ****************")
	//Iterate through the matches
	for _, match := range ns.GetMatches() {
		// All match source selector labels should be present in the requested labels map
		if !ismaglevSubset(requestConnection.GetLabels(), match.GetSourceSelector()) {
			continue
		}

		nseCandidates := []*registry.NetworkServiceEndpoint{}
		// Check all Destinations in that match
		for _, destination := range match.GetRoutes() {
			// Each NSE should be matched against that destination
			for _, nse := range networkServiceEndpoints {
				if ismaglevSubset(nse.GetLabels(), destination.GetDestinationSelector()) {
					nseCandidates = append(nseCandidates, nse)
				}

			}
		}

		if len(nseCandidates) > 0 {
			// We found candidates. Use Maglev to select one
			logrus.Infof("number of available nseCandidates = %d ", len(nseCandidates))
			// replace nil by requestConnection, because we need ReqId for Maglev decision, ReqId<=>bucket Id in lookup table
			return mg.maglev.SelectEndpoint(requestConnection, ns, nseCandidates)
			//return mg.maglev.SelectEndpoint(nil, ns, nseCandidates)
		}
	}
	logrus.Infof("return nil")
	return nil
}

func (mg *MatchMaglevSelector) SelectEndpoint(requestConnection *connection.Connection, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	logrus.Infof("Selecting maglev endpoint for requestConnection %s numMatches %d ", requestConnection.GetId(), len(ns.GetMatches()))
	if len(ns.GetMatches()) == 0 {
		//return mg.maglev.SelectEndpoint(nil, ns, networkServiceEndpoints)
		// replace nil by requestConnection, because we need ReqId for Maglev decision, ReqId<=>bucket Id in lookup table
		return mg.maglev.SelectEndpoint(requestConnection, ns, networkServiceEndpoints)
	}

	return mg.maglevEndpoint(requestConnection, ns, networkServiceEndpoints)
}
