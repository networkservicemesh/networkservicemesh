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

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

func Test_roundRobinSelector_SelectEndpoint(t *testing.T) {
	type args struct {
		requestConnection       *connection.Connection
		ns                      *registry.NetworkService
		networkServiceEndpoints []*registry.NetworkServiceEndpoint
	}
	tests := []struct {
		name string
		args args
		want *registry.NetworkServiceEndpoint
	}{
		{
			name: "network-service-1 first pass",
			args: args{
				ns: &registry.NetworkService{
					Name: "network-service-1",
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "NSE-1",
					},
					{
						Name: "NSE-2",
					},
					{
						Name: "NSE-3",
					},
				},
			},
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-1",
			},
		},
		{
			name: "network-service-1 second pass",
			args: args{
				ns: &registry.NetworkService{
					Name: "network-service-1",
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "NSE-1",
					},
					{
						Name: "NSE-2",
					},
					{
						Name: "NSE-3",
					},
				},
			},
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
			},
		},
		{
			name: "network-service-1 remove NSE-1 first pass",
			args: args{
				ns: &registry.NetworkService{
					Name: "network-service-1",
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "NSE-2",
					},
					{
						Name: "NSE-3",
					},
				},
			},
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
			},
		},
		{
			name: "network-service-1 remove NSE-1 second pass",
			args: args{
				ns: &registry.NetworkService{
					Name: "network-service-1",
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "NSE-2",
					},
					{
						Name: "NSE-3",
					},
				},
			},
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-3",
			},
		},
		{
			name: "network-service-2 first pass",
			args: args{
				ns: &registry.NetworkService{
					Name: "network-service-2",
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "NSE-2",
					},
					{
						Name: "NSE-3",
					},
					{
						Name: "NSE-1",
					},
				},
			},
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
			},
		},
		{
			name: "network-service-2 second pass",
			args: args{
				ns: &registry.NetworkService{
					Name: "network-service-2",
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "NSE-2",
					},
					{
						Name: "NSE-3",
					},
					{
						Name: "NSE-1",
					},
				},
			},
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-3",
			},
		},
		{
			name: "network-service-2 remove NSE-3 first pass",
			args: args{
				ns: &registry.NetworkService{
					Name: "network-service-2",
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "NSE-2",
					},
					{
						Name: "NSE-1",
					},
				},
			},
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
			},
		},
		{
			name: "network-service-2 remove NSE-3 second pass",
			args: args{
				ns: &registry.NetworkService{
					Name: "network-service-2",
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "NSE-2",
					},
					{
						Name: "NSE-1",
					},
				},
			},
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-1",
			},
		},
		{
			name: "network-service-1 third pass",
			args: args{
				ns: &registry.NetworkService{
					Name: "network-service-1",
				},
				networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
					{
						Name: "NSE-1",
					},
					{
						Name: "NSE-2",
					},
					{
						Name: "NSE-3",
					},
				},
			},
			want: &registry.NetworkServiceEndpoint{
				Name: "NSE-2",
			},
		},
	}
	rr := NewRoundRobinSelector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rr.SelectEndpoint(tt.args.requestConnection, tt.args.ns, tt.args.networkServiceEndpoints); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("roundRobinSelector.SelectEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}
