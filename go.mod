module github.com/networkservicemesh/networkservicemesh

go 1.13

// ./scripts/switch_k8s_version.sh to change k8s version

replace (
	git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	gonum.org/v1/gonum => github.com/gonum/gonum v0.0.0-20190331200053-3d26580ed485
	gonum.org/v1/netlib => github.com/gonum/netlib v0.0.0-20190331212654-76723241ea4e
	k8s.io/api => k8s.io/api v0.17.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.1
	k8s.io/apiserver => k8s.io/apiserver v0.17.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.1
	k8s.io/client-go => k8s.io/client-go v0.17.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.1
	k8s.io/code-generator => k8s.io/code-generator v0.17.1
	k8s.io/component-base => k8s.io/component-base v0.17.1
	k8s.io/cri-api => k8s.io/cri-api v0.17.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.1
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.1
	k8s.io/kubectl => k8s.io/kubectl v0.17.1
	k8s.io/kubelet => k8s.io/kubelet v0.17.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.1
	k8s.io/metrics => k8s.io/metrics v0.17.1
	k8s.io/node-api => k8s.io/node-api v0.17.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.1
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.1
	k8s.io/sample-controller => k8s.io/sample-controller v0.17.1
)

replace (
	github.com/networkservicemesh/networkservicemesh => ./
	github.com/networkservicemesh/networkservicemesh/applications/federation-server => ./applications/federation-server
	github.com/networkservicemesh/networkservicemesh/applications/nsmrs => ./applications/nsmrs
	github.com/networkservicemesh/networkservicemesh/controlplane => ./controlplane
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ./controlplane/api
	github.com/networkservicemesh/networkservicemesh/forwarder => ./forwarder
	github.com/networkservicemesh/networkservicemesh/forwarder/api => ./forwarder/api
	github.com/networkservicemesh/networkservicemesh/k8s => ./k8s
	github.com/networkservicemesh/networkservicemesh/k8s/api => ./k8s/api
	github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis => ./k8s/pkg/apis
	github.com/networkservicemesh/networkservicemesh/pkg => ./pkg
	github.com/networkservicemesh/networkservicemesh/scripts/aws => ./scripts/aws
	github.com/networkservicemesh/networkservicemesh/sdk => ./sdk
	github.com/networkservicemesh/networkservicemesh/side-cars => ./side-cars
	github.com/networkservicemesh/networkservicemesh/test => ./test
	github.com/networkservicemesh/networkservicemesh/utils => ./utils
)

require (
	github.com/networkservicemesh/networkservicemesh/scripts/aws v0.0.0-00010101000000-000000000000 // indirect
	github.com/networkservicemesh/networkservicemesh/test v0.0.0-00010101000000-000000000000 // indirect
)
