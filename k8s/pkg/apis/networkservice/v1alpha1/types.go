package v1alpha1

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type State string

const (
	OFFLINE = "OFFLINE"
	RUNNING = "RUNNING"
	PAUSED  = "PAUSED"
	ERROR   = "ERROR"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkService struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkServiceSpec   `json:"spec"`
	Status NetworkServiceStatus `json:"status"`
}

type NetworkServiceSpec struct {
	Payload string   `json:"payload"`
	Matches []*Match `json:"matches"`
}

type Match struct {
	SourceSelector map[string]string `json:"sourceSelector,omitempty"`
	Routes         []*Destination    `json:"route"`
}

type Destination struct {
	DestinationSelector map[string]string `json:"destinationSelector,omitempty"`
	Weight              uint32            `json:"weight,omitempty"`
}

type NetworkServiceStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkService `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceEndpoint struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkServiceEndpointSpec   `json:"spec"`
	Status NetworkServiceEndpointStatus `json:"status"`
}

type NetworkServiceEndpointSpec struct {
	NetworkServiceName string `json:"networkservicename"`
	Payload            string `json:"payload"`
	NsmName            string `json:"nsmname"`
}

type NetworkServiceEndpointStatus struct {
	State State `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceEndpointList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkServiceEndpoint `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceManager struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkServiceManagerSpec   `json:"spec"`
	Status NetworkServiceManagerStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceManagerList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkServiceManager `json:"items"`
}

type NetworkServiceManagerSpec struct {
	URL            string      `json:"url"`
	ExpirationTime metaV1.Time `json:"expirationtime"`
}

type NetworkServiceManagerStatus struct {
	State State `json:"state"`
}
