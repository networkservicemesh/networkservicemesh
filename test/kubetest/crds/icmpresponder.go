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
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nsapiv1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
)

// IcmpResponder creates a NetworkService
func IcmpResponder(sourceSelector, destinationSelector map[string]string) *nsapiv1.NetworkService {
	ns := &nsapiv1.NetworkService{
		TypeMeta: v12.TypeMeta{
			APIVersion: "networkservicemesh.io/v1alpha1",
			Kind:       "NetworkService",
		},
		ObjectMeta: v12.ObjectMeta{
			Name: "icmp-responder",
		},
		Spec: nsapiv1.NetworkServiceSpec{
			Payload: "IP",
			Matches: []*nsapiv1.Match{
				{
					SourceSelector: sourceSelector,
					Routes: []*nsapiv1.Destination{
						{
							DestinationSelector: destinationSelector,
						},
					},
				},
			},
		},
	}
	return ns
}
