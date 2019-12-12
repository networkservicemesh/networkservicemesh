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
	"testing"
	"github.com/sirupsen/logrus"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

func Test_MaglevSelector_SelectEndpoint(t *testing.T) {
	type args struct {
		requestConnection       *connection.Connection
		ns                      *registry.NetworkService
		networkServiceEndpoints []*registry.NetworkServiceEndpoint
	}
	tests := []struct {
		name string
		args Maglevargs
		want *registry.NetworkServiceEndpoint
	}{
		{
			name: "network-service-1 first pass",
			args: genMaglevArgs(
				genArgsMagParam{
				//ns: &registry.NetworkService{
					nsID:         1,
					reqId:		  "1",
				//},
				endpoints: []*registry.NetworkServiceEndpoint{
					{
						EndpointName: "NSE-1",
					},
					{
						EndpointName: "NSE-2",
					},
					{
						EndpointName: "NSE-3",
					},
				},
			}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-1",
			},
		},
		{
			name: "network-service-1 second pass",
			args: genMaglevArgs(
				genArgsMagParam{
				//ns: &registry.NetworkService{
					nsID:         1,
					reqId:		  "2",
				//},
				endpoints: []*registry.NetworkServiceEndpoint{
					{
						EndpointName: "NSE-1",
					},
					{
						EndpointName: "NSE-2",
					},
					{
						EndpointName: "NSE-3",
					},
				},
			}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
			},
		},
		{
			name: "network-service-1 remove NSE-1 first pass",
			args: genMaglevArgs(
				genArgsMagParam{
				//ns: &registry.NetworkService{
					nsID:         1,
					reqId:		  "3",
				//},
				endpoints: []*registry.NetworkServiceEndpoint{
					{
						EndpointName: "NSE-2",
					},
					{
						EndpointName: "NSE-3",
					},
				},
			}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
			},
		},
		{
			name: "network-service-1 remove NSE-1 second pass",
			args: genMaglevArgs(
				genArgsMagParam{
				//ns: &registry.NetworkService{
					nsID:         1,
					reqId:		  "4",
				//},
				endpoints: []*registry.NetworkServiceEndpoint{
					{
						EndpointName: "NSE-2",
					},
					{
						EndpointName: "NSE-3",
					},
				},
			}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-3",
			},
		},
		{
			name: "network-service-2 first pass",
			args: genMaglevArgs(
				genArgsMagParam{
				//ns: &registry.NetworkService{
					nsID:         2,
					reqId:		  "5",
				//},
				endpoints: []*registry.NetworkServiceEndpoint{
					{
						EndpointName: "NSE-2",
					},
					{
						EndpointName: "NSE-3",
					},
					{
						EndpointName: "NSE-1",
					},
				},
			}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
			},
		},
		{
			name: "network-service-2 second pass",
			args: genMaglevArgs(
				genArgsMagParam{
				//ns: &registry.NetworkService{
					nsID:         2,
					reqId:		  "6",
				//},
				endpoints: []*registry.NetworkServiceEndpoint{
					{
						EndpointName: "NSE-2",
					},
					{
						EndpointName: "NSE-3",
					},
					{
						EndpointName: "NSE-1",
					},
				},
			}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-3",
			},
		},
		{
			name: "network-service-2 remove NSE-3 first pass",
			args: genMaglevArgs(
				genArgsMagParam{
				//ns: &registry.NetworkService{
					nsID:         2,
					reqId:		  "7",
				//},
				endpoints: []*registry.NetworkServiceEndpoint{
					{
						EndpointName: "NSE-2",
					},
					{
						EndpointName: "NSE-1",
					},
				},
			}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
			},
		},
		{
			name: "network-service-2 remove NSE-3 second pass",
			args: genMaglevArgs(
				genArgsMagParam{
				//ns: &registry.NetworkService{
					nsID:         2,
					reqId:		  "8",
				//},
				endpoints: []*registry.NetworkServiceEndpoint{
					{
						EndpointName: "NSE-2",
					},
					{
						EndpointName: "NSE-1",
					},
				},
			}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-1",
			},
		},
		{
			name: "network-service-1 third pass",
			args: genMaglevArgs(
				genArgsMagParam{
				//ns: &registry.NetworkService{
					nsID:         1,
					reqId:		  "9",
				//},
				endpoints: []*registry.NetworkServiceEndpoint{
					{
						EndpointName: "NSE-1",
					},
					{
						EndpointName: "NSE-2",
					},
					{
						EndpointName: "NSE-3",
					},
				},
			}),
			want: &registry.NetworkServiceEndpoint{
				EndpointName: "NSE-2",
			},
		},
	}
	mg := NewmaglevSelector()
	//NewMatchMaglevSelector()

	for _, tt := range tests {
		
		t.Run(tt.name, func(t *testing.T) {
			//logrus.Infof("test for tt name %s nse %v ", tt.name, tt.args.networkServiceEndpoints)
			logrus.Infof("test for tt name %s and requestconnection %s ", tt.name, tt.args.requestConnection.GetId())
			if got := mg.SelectEndpoint(tt.args.requestConnection, tt.args.ns, tt.args.networkServiceEndpoints); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("maglevSelector.SelectEndpoint() = %v, want %v name %s ", got, tt.want, tt.args.ns.GetName())
			}
		})
	}
}
