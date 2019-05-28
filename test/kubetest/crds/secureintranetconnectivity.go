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

package crds

import (
	"strconv"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nsapiv1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
)

// SecureIntranetConnectivity creates a NetworkService
func SecureIntranetConnectivity(ptnum int) *nsapiv1.NetworkService {
	ns := &nsapiv1.NetworkService{
		TypeMeta: v12.TypeMeta{
			APIVersion: "networkservicemesh.io/v1alpha1",
			Kind:       "NetworkService",
		},
		ObjectMeta: v12.ObjectMeta{
			Name: "secure-intranet-connectivity",
		},
		Spec: nsapiv1.NetworkServiceSpec{
			Payload: "IP",
			Matches: []*nsapiv1.Match{
				&nsapiv1.Match{
					SourceSelector: map[string]string{
						"app": "firewall",
					},
					Routes: []*nsapiv1.Destination{
						&nsapiv1.Destination{
							DestinationSelector: map[string]string{
								"app": "vpn-gateway",
							},
						},
					},
				},
				&nsapiv1.Match{
					Routes: []*nsapiv1.Destination{
						&nsapiv1.Destination{
							DestinationSelector: map[string]string{
								"app": "firewall",
							},
						},
					},
				},
			},
		},
	}
	matches := ns.Spec.Matches
	for i := 0; i < ptnum; i++ {
		id := strconv.Itoa(i + 1)
		dest := matches[i].Routes[0].DestinationSelector["app"]
		matches[i].Routes[0].DestinationSelector["app"] = "passthrough-" + id

		m := &nsapiv1.Match{
			SourceSelector: map[string]string{
				"app": "passthrough-" + id,
			},
			Routes: []*nsapiv1.Destination{
				&nsapiv1.Destination{
					DestinationSelector: map[string]string{
						"app": dest,
					},
				},
			},
		}

		t := append([]*nsapiv1.Match{m}, matches[i+1:]...)
		matches = append(matches[:i+1], t...)
	}
	ns.Spec.Matches = matches
	return ns
}
