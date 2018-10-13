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

package v1

import (
	"reflect"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
)

// Constants to register CRDs for our resources
const (
	NSMGroup        = "networkservicemesh.io"
	NSMGroupVersion = "v1"
	NSMSpecPath     = "github.com/ligato/networkservicemesh/pkg/apis/" + NSMGroup
	NSMEPSingular   = "networkserviceendpoint"
	NSMEPPlural     = NSMEPSingular + "s"
	NSMSingular     = "networkservice"
	NSMPlural       = NSMSingular + "s"
)

var (
	// NSMEPTypeName is the reflected type name of a NetworkServiceEndpoint
	NSMEPTypeName = reflect.TypeOf(NetworkServiceEndpoint{}).Name()
	// FullNSMEPName is the full name (including path) of NetworkServiceEndpoints
	FullNSMEPName = NSMSpecPath + "/" + NSMGroupVersion + "." + NSMEPTypeName
	// NSMTypeName is the reflected type name of a NetworkService
	NSMTypeName = reflect.TypeOf(NetworkService{}).Name()
	// FullNSMName is the full name (including path) of NetworkServices
	FullNSMName = NSMSpecPath + "/" + NSMGroupVersion + "." + NSMTypeName
)

// NetworkServiceEndpoint CRD
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type NetworkServiceEndpoint struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata,omitempty"`
	Spec            *netmesh.NetworkServiceEndpoint `json:"spec"`
	Status          NetworkServiceEndpointStatus    `json:"status,omitempty"`
}

//NetworkServiceEndpointStatus is the status schema for this CRD
type NetworkServiceEndpointStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

// NetworkServiceEndpointList is the list schema for this CRD
// -genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type NetworkServiceEndpointList struct {
	meta.TypeMeta `json:",inline"`
	// +optional
	meta.ListMeta `json:"metadata,omitempty"`
	Items         []NetworkServiceEndpoint `json:"items"`
}

// NetworkService CRD
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type NetworkService struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata,omitempty"`
	Spec            *netmesh.NetworkService `json:"spec"`
	Status          NetworkServiceStatus    `json:"status,omitempty"`
}

// NetworkServiceStatus is the status schema for this CRD
type NetworkServiceStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

// NetworkServiceList is the list schema for this CRD
// -genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type NetworkServiceList struct {
	meta.TypeMeta `json:",inline"`
	// +optional
	meta.ListMeta `json:"metadata,omitempty"`
	Items         []NetworkService `json:"items"`
}
