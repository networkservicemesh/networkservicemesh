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
type NetworkServiceStatus struct {
	URIs []string `json:"uris"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkServiceList struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata,omitempty"`

	Items []NetworkService `json:"items"`
}
