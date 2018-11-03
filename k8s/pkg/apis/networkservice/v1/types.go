package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkServiceSpec   `json:"spec"`
	Status NetworkServiceStatus `json:"status"`
}

type NetworkServiceSpec struct {
	Payload string `json:"payload"`
}

type NetworkServiceStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkService `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceEndpoint struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkServiceEndpointSpec   `json:"spec"`
	Status NetworkServiceEndpointStatus `json:"status"`
}

type NetworkServiceEndpointSpec struct {
	NetworkServiceName string `json:"networkservicename"`
	NsmName            string `json:"nsmname"`
}

type NetworkServiceEndpointStatus struct {
	LastSeen meta_v1.Time `json:"lastseen"`
	State    State        `json:"state"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceEndpointList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkServiceEndpoint `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceManager struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkServiceManagerSpec   `json:"spec"`
	Status NetworkServiceManagerStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceManagerList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata,omitempty"`

	Items []NetworkServiceManager `json:"items"`
}

type NetworkServiceManagerSpec struct {
}

type NetworkServiceManagerStatus struct {
	LastSeen meta_v1.Time `json:"lastseen"`
	URL      string       `json:"url"`
	State    State        `json:"state"`
}
