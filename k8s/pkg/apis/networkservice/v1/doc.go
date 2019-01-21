// +k8s:deepcopy-gen=package
// +groupName=networkservicemesh.io
//go:generate deepcopy-gen --go-header-file ../../../../../conf/boilerplate2.txt -O zz_generated.deepcopy
//go:generate ../../../../../vendor/k8s.io/code-generator/generate-groups.sh all github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis networkservice:v1  --output-base ../../../../../../../../ --go-header-file ../../../../../conf/boilerplate2.txt
package v1
