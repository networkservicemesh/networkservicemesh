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
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type args struct {
	requestConnection       *connection.Connection
	ns                      *registry.NetworkService
	networkServiceEndpoints []*registry.NetworkServiceEndpoint
}

type genArgsParam struct {
	nsID, numMatches, numEndpoints int
	labelIDs                       []int
	matchSourceSelector            map[int]map[string]string
	matchDestinationSelector       map[int]map[string]string
	endpoints                      []*registry.NetworkServiceEndpoint
}

func genArgs(p genArgsParam) args {
	labels := map[string]string{}
	for _, id := range p.labelIDs {
		labels["label"+strconv.Itoa(id)] = "value" + strconv.Itoa(id)
	}
	matches := []*registry.Match{}

	for i := 1; i <= p.numMatches; i++ {
		sourceSelector := map[string]string{
			"label" + strconv.Itoa(i): "value" + strconv.Itoa(i),
		}
		if _, ok := p.matchSourceSelector[i]; ok {
			sourceSelector = p.matchSourceSelector[i]
		}
		routes := []*registry.Destination{
			{
				DestinationSelector: map[string]string{
					"label" + strconv.Itoa(i): "value" + strconv.Itoa(i),
				},
			},
		}
		if _, ok := p.matchDestinationSelector[i]; ok {
			routes = append(routes, &registry.Destination{
				DestinationSelector: p.matchDestinationSelector[i],
			})
		}
		matches = append(matches, &registry.Match{
			SourceSelector: sourceSelector,
			Routes:         routes,
		})
	}

	endpoints := []*registry.NetworkServiceEndpoint{}

	for i := 1; i <= p.numEndpoints; i++ {
		endpoints = append(endpoints, &registry.NetworkServiceEndpoint{
			Name: "NSE-" + strconv.Itoa(i),
			Labels: map[string]string{
				"label" + strconv.Itoa(i): "value" + strconv.Itoa(i),
			},
		})
	}
	endpoints = append(endpoints, p.endpoints...)

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
		name string
		args args
		want *registry.NetworkServiceEndpoint
	}{
		{
			name: "network-service-1 RR fallback",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					numMatches:   0,
					numEndpoints: 3,
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-1",
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
					numMatches:   0,
					numEndpoints: 3,
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
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
					numMatches:   2,
					numEndpoints: 3,
					labelIDs:     []int{1},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-1",
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
					numMatches:   3,
					numEndpoints: 5,
					labelIDs:     []int{2},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
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
					numMatches:   42,
					numEndpoints: 24,
					labelIDs:     []int{12},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-12",
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
					numMatches:   42,
					numEndpoints: 24,
					labelIDs:     []int{42},
				}),
			want: nil,
		},
		{
			name: "network-service-1 label1 label2",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					labelIDs:     []int{1, 2},
					matchSourceSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					matchDestinationSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					endpoints: []*registry.NetworkServiceEndpoint{
						{
							Name: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
		},
		{
			name: "network-service-1 label1 label2 pass2",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					labelIDs:     []int{1, 2},
					matchSourceSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					matchDestinationSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					endpoints: []*registry.NetworkServiceEndpoint{
						{
							Name: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-1",
				Labels: map[string]string{
					"label1": "value1",
				},
			},
		},
		{
			name: "network-service-1 label1 label2 pass3",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					labelIDs:     []int{1, 2},
					matchSourceSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					matchDestinationSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					endpoints: []*registry.NetworkServiceEndpoint{
						{
							Name: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
		},
		{
			name: "network-service-1 label2 label1",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					labelIDs:     []int{2, 1},
					matchSourceSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					matchDestinationSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					endpoints: []*registry.NetworkServiceEndpoint{
						{
							Name: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-1",
				Labels: map[string]string{
					"label1": "value1",
				},
			},
		},
		{
			name: "network-service-1 label2 label1 pass2",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					labelIDs:     []int{2, 1},
					matchSourceSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					matchDestinationSelector: map[int]map[string]string{
						2: {
							"label1": "value1",
							"label2": "value2",
						},
					},
					endpoints: []*registry.NetworkServiceEndpoint{
						{
							Name: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
		},
		{
			name: "network-service-1 label10 label20 label30",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					numMatches:   100,
					numEndpoints: 500,
					labelIDs:     []int{10, 20, 30},
					matchSourceSelector: map[int]map[string]string{
						10: {
							"label10": "value10",
							"label20": "value20",
							"label30": "value30",
						},
					},
					matchDestinationSelector: map[int]map[string]string{
						10: {
							"label221": "value1",
							"label222": "value2",
						},
					},
					endpoints: []*registry.NetworkServiceEndpoint{
						{
							Name: "NSE-221",
							Labels: map[string]string{
								"label221": "value1",
								"label222": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-10",
				Labels: map[string]string{
					"label10": "value10",
				},
			},
		},
		{
			name: "network-service-1 label10 label20 label30 pass2",
			args: genArgs(
				genArgsParam{
					nsID:         1,
					numMatches:   100,
					numEndpoints: 500,
					labelIDs:     []int{10, 20, 30},
					matchSourceSelector: map[int]map[string]string{
						10: {
							"label10": "value10",
							"label20": "value20",
							"label30": "value30",
						},
					},
					matchDestinationSelector: map[int]map[string]string{
						10: {
							"label221": "value1",
							"label222": "value2",
						},
					},
					endpoints: []*registry.NetworkServiceEndpoint{
						{
							Name: "NSE-221",
							Labels: map[string]string{
								"label221": "value1",
								"label222": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-221",
				Labels: map[string]string{
					"label221": "value1",
					"label222": "value2",
				},
			},
		},
		{
			name: "match any with non-empty source selector",
			args: args{
				requestConnection: &connection.Connection{
					Labels: map[string]string{
						"app": "firewall",
					},
				},
				ns: &registry.NetworkService{
					Name: "test-ns",
					Matches: []*registry.Match{
						{
							SourceSelector: map[string]string{
								"app": "firewall",
							},
							Routes: []*registry.Destination{
								{
									DestinationSelector: map[string]string{
										"app": "passthrough-1",
									},
								},
							},
						},
						{
							Routes: []*registry.Destination{
								{
									DestinationSelector: map[string]string{
										"app": "firewall",
									},
								},
							},
						},
					},
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "firewall",
						Labels: map[string]string{
							"app": "firewall",
						},
					},
				},
			},
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
