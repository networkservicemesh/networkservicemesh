// Copyright (c) 2018 Cisco and/or its affiliates.
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

package netmeshplugincrd

import (
	v1 "github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// API is the interface to a CRD plugin
type API interface {
	ListNetworkServices(selector labels.Selector) (ret []*v1.NetworkService, err error)
	ListNetworkServiceChannels(selector labels.Selector) (ret []*v1.NetworkServiceChannel, err error)
	ListNetworkServiceEndpoints(selector labels.Selector) (ret []*v1.NetworkServiceEndpoint, err error)
}
