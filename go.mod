module github.com/networkservicemesh/networkservicemesh

replace (
	// ./scripts/switch_k8s_version.sh to change k8s version
	git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	gonum.org/v1/gonum => github.com/gonum/gonum v0.0.0-20190331200053-3d26580ed485
	gonum.org/v1/netlib => github.com/gonum/netlib v0.0.0-20190331212654-76723241ea4e
	k8s.io/api => k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191114110141-0a35778df828
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191114112024-4bbba8331835
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191114111741-81bb9acf592d
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191114102325-35a9586014f7
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191114112310-0da609c4ca2d
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191114103820-f023614fb9ea
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191114111510-6d1ed697a64b
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191114110717-50a77e50d7d9
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191114111229-2e90afcb56c7
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191114110954-d67a8e7e2200
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191114112655-db9be3e678bb
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191114105837-a4a2842dc51b
	k8s.io/node-api => k8s.io/node-api v0.0.0-20191114112948-fde05759caf8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191114104439-68caf20693ac
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20191114110435-31b16e91580f
	k8s.io/sample-controller => k8s.io/sample-controller v0.0.0-20191114104921-b2770fad52e3
)

replace (
	github.com/networkservicemesh/networkservicemesh => ./
	github.com/networkservicemesh/networkservicemesh/controllers/sriov-controller => ./controllers/sriov-controller
	github.com/networkservicemesh/networkservicemesh/controlplane => ./controlplane
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ./controlplane/api
	github.com/networkservicemesh/networkservicemesh/forwarder => ./forwarder
	github.com/networkservicemesh/networkservicemesh/forwarder/api => ./forwarder/api
	github.com/networkservicemesh/networkservicemesh/k8s => ./k8s
	github.com/networkservicemesh/networkservicemesh/k8s/api => ./k8s/api
	github.com/networkservicemesh/networkservicemesh/pkg => ./pkg
	github.com/networkservicemesh/networkservicemesh/scripts/aws => ./scripts/aws
	github.com/networkservicemesh/networkservicemesh/sdk => ./sdk
	github.com/networkservicemesh/networkservicemesh/side-cars => ./side-cars
	github.com/networkservicemesh/networkservicemesh/test => ./test
	github.com/networkservicemesh/networkservicemesh/utils => ./utils
)

go 1.13

require (
	github.com/networkservicemesh/networkservicemesh/controllers/sriov-controller v0.0.0-00010101000000-000000000000 // indirect
	github.com/networkservicemesh/networkservicemesh/scripts/aws v0.0.0-00010101000000-000000000000 // indirect
	github.com/networkservicemesh/networkservicemesh/test v0.0.0-00010101000000-000000000000 // indirect
)
