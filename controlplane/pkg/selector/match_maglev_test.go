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
	//"context"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	//"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	//"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

type Maglevargs struct {
	requestConnection       *connection.Connection
	ns                      *registry.NetworkService
	networkServiceEndpoints []*registry.NetworkServiceEndpoint
	// added for getEdnpoint test
	//IgnoreEndpoints 		map[string]*registry.NSERegistration
}

type genArgsMagParam struct {
	nsID, numMatches, numEndpoints int
	reqId 						   string
	labelIDs                       []int
	matchSourceSelector            map[int]map[string]string
	matchDestinationSelector       map[int]map[string]string
	endpoints                      []*registry.NetworkServiceEndpoint
	
}

func genMaglevArgs(p genArgsMagParam) Maglevargs {

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

	//ignoreEndpoints := map[string]*registry.NSERegistration{}

	for i := 1; i <= p.numEndpoints; i++ {
		endpoints = append(endpoints, &registry.NetworkServiceEndpoint{
			EndpointName: "NSE-" + strconv.Itoa(i),
			Labels: map[string]string{
				"label" + strconv.Itoa(i): "value" + strconv.Itoa(i),
			},
		})
		/*ignoreEndpoints["NSE-" + strconv.Itoa(i)] = &registry.NetworkServiceEndpoint{
			EndpointName: "NSE-" + strconv.Itoa(i),
			Labels: map[string]string{
				"label" + strconv.Itoa(i): "value" + strconv.Itoa(i),
			},}*/
	}
	endpoints = append(endpoints, p.endpoints...)
	

	logrus.Infof("current maglev req id %s ", p.reqId)
	return Maglevargs{
		requestConnection: &connection.Connection{
			Id: 	p.reqId,
			Labels: labels,
		},
		ns: &registry.NetworkService{
			Name:    "network-service-" + strconv.Itoa(p.nsID),
			Matches: matches,
		},
		networkServiceEndpoints: endpoints,
		//IgnoreEndpoints: ignoreEndpoints,
	}
}

func Test_maglevSelector_SelectEndpoint(t *testing.T) {

	tests := []struct {
		name string
		args Maglevargs
		want *registry.NetworkServiceEndpoint
	}{
		{
			name: "network-service-1 RR fallback",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   0,
					numEndpoints: 3,
					reqId:		  "1",
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
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   0,
					numEndpoints: 3,
					reqId:		  "2",
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
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   1,
					numEndpoints: 3,
					reqId:		  "3",
				}),
			want: nil,
		},
		{
			name: "network-service-1 label1",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 3,
					reqId:		  "4",
					labelIDs:     []int{1},
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
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   3,
					numEndpoints: 5,
					reqId:		  "5",
					labelIDs:     []int{2},
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
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   42,
					numEndpoints: 24,
					reqId:		  "6",
					labelIDs:     []int{12},
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
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   42,
					numEndpoints: 24,
					reqId:		  "7",
					labelIDs:     []int{42},
				}),
			want: nil,
		},
		{
			name: "network-service-1 label1 label2",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					reqId:		  "8",
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
							EndpointName: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
		},
		{
			name: "network-service-1 label1 label2",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					reqId:		  "9",
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
							EndpointName: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
		},
		{
			name: "network-service-1 label1 label2 pass2",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					reqId:		  "10",
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
							EndpointName: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-1",
				Labels: map[string]string{
					"label1": "value1",
				},
			},
		},
		{
			name: "network-service-1 label1 label2 pass3",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					reqId:		  "11",
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
							EndpointName: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
		},
		{
			name: "network-service-1 label2 label1",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					reqId:		  "12",
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
							EndpointName: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-1",
				Labels: map[string]string{
					"label1": "value1",
				},
			},
		},
		{
			name: "network-service-1 label2 label1 pass2",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   2,
					numEndpoints: 2,
					reqId:		  "13",
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
							EndpointName: "NSE-2",
							Labels: map[string]string{
								"label1": "value1",
								"label2": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
				Labels: map[string]string{
					"label1": "value1",
					"label2": "value2",
				},
			},
		},
		{
			name: "network-service-1 label10 label20 label30",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   100,
					numEndpoints: 500,
					reqId:		  "14",
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
							EndpointName: "NSE-221",
							Labels: map[string]string{
								"label221": "value1",
								"label222": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-10",
				Labels: map[string]string{
					"label10": "value10",
				},
			},
		},
		{
			name: "network-service-1 label10 label20 label30 pass2",
			args: genMaglevArgs(
				genArgsMagParam{
					nsID:         1,
					numMatches:   100,
					numEndpoints: 500,
					reqId:		  "15",
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
							EndpointName: "NSE-221",
							Labels: map[string]string{
								"label221": "value1",
								"label222": "value2",
							},
						},
					},
				}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-221",
				Labels: map[string]string{
					"label221": "value1",
					"label222": "value2",
				},
			},
		},
	}

	
	mg := NewMatchMaglevSelector()

	// GetEndpoint testing
	/*serviceRegistry := nsmd.NewServiceRegistry()
	model := model.NewModel() 

	properties := nsm.NewNsmProperties()
	nseManager := &nseManager{
		serviceRegistry: serviceRegistry,
		model:           model,
		properties:      properties,
	}*/
	
	for _, tt := range tests {
		logrus.Infof("test Maglev selector for ns req %v reqId %s and ns %v ", tt.args.requestConnection, tt.args.requestConnection.GetId(), tt.args.ns.GetName())
		//mg.GetRequestName(tt.args.requestConnection.GetId())
		t.Run(tt.name, func(t *testing.T) {
			logrus.Infof("tt.args.requestConnection %v ", tt.args.requestConnection)
			if got := mg.SelectEndpoint(tt.args.requestConnection, tt.args.ns, tt.args.networkServiceEndpoints); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("matchMaglevSelector.SelectEndpoint() = %v, want %v, name %s tt.args.ns.name %s args.getReqId %s ", got, tt.want, tt.name, tt.args.ns.GetName(), tt.args.requestConnection.GetId())
			}
			//nseManager.getEndpoint(context.Background(), tt.args.requestConnection, tt.args.IgnoreEndpoints)
		})
	}
}
