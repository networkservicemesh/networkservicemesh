package v1

import (
	"github.com/ligato/networkservicemesh/k8s/pkg/apis/networkservice"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   networkservice.GroupName,
	Version: "v1",
}

var scheme = runtime.NewSchemeBuilder(addKnownTypes)
var AddToScheme = scheme.AddToScheme

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds our types to the API scheme by registering
// NetworkService and NetworkServiceList
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&NetworkService{},
		&NetworkServiceList{},
		&NetworkServiceEndpoint{},
		&NetworkServiceEndpointList{},
	)

	// register the type in the scheme
	meta_v1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
