// Copyright 2019 VMware, Inc.
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
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
)

type fields struct {
	Mutex      sync.Mutex
	roundRobin Selector
}
type args struct {
	requestConnection       *connection.Connection
	ns                      *registry.NetworkService
	networkServiceEndpoints []*registry.NetworkServiceEndpoint
}

type genArgsParam struct {
	nsID, labelID, numMatches, numEndpoints int
}

func genArgs(p genArgsParam) args {

	labels := map[string]string{}
	if p.labelID > 0 {
		labels["label"+strconv.Itoa(p.labelID)] = "value" + strconv.Itoa(p.labelID)
	}
	matches := []*registry.Match{}

	for i := 1; i <= p.numMatches; i++ {
		matches = append(matches, &registry.Match{
			SourceSelector: map[string]string{
				"label" + strconv.Itoa(i): "value" + strconv.Itoa(i),
			},
			Routes: []*registry.Destination{
				{
					DestinationSelector: map[string]string{
						"label" + strconv.Itoa(i): "value" + strconv.Itoa(i),
					},
				},
			},
		})
	}

	endpoints := []*registry.NetworkServiceEndpoint{}

	for i := 1; i <= p.numEndpoints; i++ {
		endpoints = append(endpoints, &registry.NetworkServiceEndpoint{
			EndpointName: "NSE-" + strconv.Itoa(i),
			Labels: map[string]string{
				"label" + strconv.Itoa(i): "value" + strconv.Itoa(i),
			},
		})
	}

	return args{
		requestConnection: &connection.Connection{
			Labels: labels,
		},
		ns: &registry.NetworkService{
			Name:    "network-service-" + strconv.Itoa(p.nsID),
			Matches: matches,
		},
		networkServiceEndpoints: endpoints,
	}
}

func Test_matchSelector_SelectEndpoint(t *testing.T) {

	tests := []struct {
		name   string
		fields fields
		args   args
		want   *registry.NetworkServiceEndpoint
	}{
		{
			name: "network-service-1 RR fallback",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					labelID:      0,
					numMatches:   0,
					numEndpoints: 3,
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-1",
				Labels: map[string]string{
					"label1": "value1",
				},
			},
		},
		{
			name: "network-service-1 RR fallback second pass",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					labelID:      0,
					numMatches:   0,
					numEndpoints: 3,
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
				Labels: map[string]string{
					"label2": "value2",
				},
			},
		},
		{
			name: "network-service-1 no labels",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					labelID:      0,
					numMatches:   1,
					numEndpoints: 3,
				}),
			want: nil,
		},
		{
			name: "network-service-1 label1",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					labelID:      1,
					numMatches:   2,
					numEndpoints: 3,
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-1",
				Labels: map[string]string{
					"label1": "value1",
				},
			},
		},
		{
			name: "network-service-1 label2",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					labelID:      2,
					numMatches:   3,
					numEndpoints: 5,
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
				Labels: map[string]string{
					"label2": "value2",
				},
			},
		},
		{
			name: "network-service-1 label12",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					labelID:      12,
					numMatches:   42,
					numEndpoints: 24,
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-12",
				Labels: map[string]string{
					"label12": "value12",
				},
			},
		},
		{
			name: "network-service-1 non existing",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					labelID:      42,
					numMatches:   42,
					numEndpoints: 24,
				}),
			want: nil,
		},
	}
	m := NewMatchSelector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.SelectEndpoint(tt.args.requestConnection, tt.args.ns, tt.args.networkServiceEndpoints); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("matchSelector.SelectEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}
