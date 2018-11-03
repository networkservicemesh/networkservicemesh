package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	State string `json:"state"`
}

type NetworkServiceEndpointList struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	Items []NetworkServiceEndpoint `json:"items"`
}

type NetworkServiceManager struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkServiceManagerSpec   `json:"spec"`
	Status NetworkServiceManagerStatus `json:"status"`
}

type NetworkServiceManagerList struct {
	Items []NetworkServiceManager `json:"items"`
}

type NetworkServiceManagerSpec struct {
}
type NetworkServiceManagerStatus struct {
	LastSeen uint64 `json:"lastseen"`
	URL      string `json:"url"`
}
