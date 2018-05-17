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
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ligato/networkservicemesh/netmesh/model/netmesh"
)

// Constants to register CRDs for our resources
const (
	NSMGroup        string = "networkservicemesh.io"
	NSMGroupVersion string = "v1"

	NSMEPPlural   string = "networkserviceendpoints"
	FullNSMEPName string = NSMEPPlural + "." + NSMGroup

	NSMChannelPlural   string = "networkservicechannels"
	FullNSMChannelName string = NSMChannelPlural + "." + NSMGroup

	NSMPlural   string = "networkservices"
	FullNSMName string = NSMPlural + "." + NSMGroup
)

// NetworkServiceEndpoint CRD
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceEndpoint struct {
	meta_v1.TypeMeta   `json:",inline,namespace=networkserviceendpoints"`
	meta_v1.ObjectMeta `json:"metadata,omitempty,namespace=networkserviceendpoints"`
	Spec               netmesh.NetworkServiceEndpoint   `json:"spec"`
	Validation         NetworkServiceEndpointValidation `json:"validation"`
	Status             NetworkServiceEndpointStatus     `json:"status,omitempty"`
}

// NetworkServiceEndpointValidation is the schema validation for this CRD
type NetworkServiceEndpointValidation struct {
	ID   string `json:"^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$"`
	Name string `json:"^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{1,64}$"`
}

//NetworkServiceEndpointStatus is the status schema for this CRD
type NetworkServiceEndpointStatus struct {
	State   string `json:"state,omitempty"`
	Message string `kson:"message,omitempty"`
}

// NetworkServiceEndpointList is the list schema for this CRD
// -genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceEndpointList struct {
	meta_v1.TypeMeta `json:",inline"`
	// +optional
	meta_v1.ListMeta `json:"metadata,omitempty"`
	Items            []NetworkServiceEndpoint `json:"items"`
}

// NetworkServiceChannel CRD
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceChannel struct {
	meta_v1.TypeMeta   `json:",inline,namespace=networkservicechannels"`
	meta_v1.ObjectMeta `json:"metadata,omitempty,namespace=networkservicechannels"`
	Spec               netmesh.NetworkService_NetmeshChannel `json:"spec"`
	Validation         NetworkServiceChannelValidation       `json:"validation"`
	Status             NetworkServiceChannelStatus           `json:"status,omitempty"`
}

// NetworkServiceChannelValidation is the validation schema for this CRD
type NetworkServiceChannelValidation struct {
	ID      string `json:"^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$"`
	Name    string `json:"^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{1,64}$"`
	Payload string `json:"^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{1,64}$"`
}

// NetworkServiceChannelStatus is the status schema for this CRD
type NetworkServiceChannelStatus struct {
	State   string `json:"state,omitempty"`
	Message string `kson:"message,omitempty"`
}

// NetworkServiceChannelList is the list schema for this CRD
// -genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceChannelList struct {
	meta_v1.TypeMeta `json:",inline"`
	// +optional
	meta_v1.ListMeta `json:"metadata,omitempty"`
	Items            []NetworkServiceChannel `json:"items"`
}

// NetworkService CRD
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkService struct {
	meta_v1.TypeMeta   `json:",inline,namespace=networkservices"`
	meta_v1.ObjectMeta `json:"metadata,omitempty,namespace=networkservices"`
	Spec               netmesh.NetworkService   `json:"spec"`
	Validation         NetworkServiceValidation `json:"validation"`
	Status             NetworkServiceStatus     `json:"status,omitempty"`
}

// NetworkServiceValidation is the validation schema for this CRD
type NetworkServiceValidation struct {
	ID       string `json:"^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$"`
	Name     string `json:"^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{1,64}$"`
	Selector string `json:"^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{1,64}$"`
	Channels string `json:"minimum=1,maximum=10"`
}

// NetworkServiceStatus is the status schema for this CRD
type NetworkServiceStatus struct {
	State   string `json:"state,omitempty"`
	Message string `kson:"message,omitempty"`
}

// NetworkServiceList is the list schema for this CRD
// -genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceList struct {
	meta_v1.TypeMeta `json:",inline"`
	// +optional
	meta_v1.ListMeta `json:"metadata,omitempty"`
	Items            []NetworkService `json:"items"`
}
