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
	"bytes"
	"sync"
	"text/template"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
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
func isSubset(a, b, nsLabels map[string]string) bool {
	if len(a) < len(b) {
		return false
	}
	for k, v := range b {
		if a[k] != v {
			result := ProcessLabels(v, nsLabels)
			if a[k] != result {
				return false
			}
		}
	}
	return true
}

func (m *matchSelector) matchEndpoint(nsLabels map[string]string, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	logrus.Infof("Matching endpoint for labels %v", nsLabels)

	matchedNonEmptySelector := false
	//Iterate through the matches
	for _, match := range ns.GetMatches() {
		// All match source selector labels should be present in the requested labels map
		if !isSubset(nsLabels, match.GetSourceSelector(), nsLabels) {
			continue
		}

		// If we already have matched any non empty selector we shouldn't match empty selector
		if len(match.GetSourceSelector()) == 0 && matchedNonEmptySelector {
			continue
		}

		if len(match.GetSourceSelector()) != 0 {
			matchedNonEmptySelector = true
		}

		nseCandidates := []*registry.NetworkServiceEndpoint{}
		// Check all Destinations in that match
		for _, destination := range match.GetRoutes() {
			// Each NSE should be matched against that destination
			for _, nse := range networkServiceEndpoints {
				if isSubset(nse.GetLabels(), destination.GetDestinationSelector(), nsLabels) {
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
	logrus.Infof("Selecting endpoint for %s with %d matches.", requestConnection.GetNetworkService(), len(ns.GetMatches()))
	if len(ns.GetMatches()) == 0 {
		return m.roundRobin.SelectEndpoint(nil, ns, networkServiceEndpoints)
	}

	return m.matchEndpoint(requestConnection.GetLabels(), ns, networkServiceEndpoints)
}

// ProcessLabels generates matches based on destination label selectors that specify templating.
func ProcessLabels(str string, vars interface{}) string {
	tmpl, err := template.New("tmpl").Parse(str)

	if err != nil {
		panic(err)
	}
	return process(tmpl, vars)
}

func process(t *template.Template, vars interface{}) string {
	var tmplBytes bytes.Buffer

	err := t.Execute(&tmplBytes, vars)
	if err != nil {
		panic(err)
	}
	return tmplBytes.String()
}
