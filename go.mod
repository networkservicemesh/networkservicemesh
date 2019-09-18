module github.com/networkservicemesh/networkservicemesh

require (
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d // indirect
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.7.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spiffe/spire v0.0.0-20190515205011-c8123525fba8
	golang.org/x/sys v0.0.0-20190618155005-516e3c20635f // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20190611190212-a7e196e89fd3 // indirect
	google.golang.org/grpc v1.23.1 // indirect
)

replace (
	git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	gonum.org/v1/gonum => github.com/gonum/gonum v0.0.0-20190331200053-3d26580ed485
	gonum.org/v1/netlib => github.com/gonum/netlib v0.0.0-20190331212654-76723241ea4e
	k8s.io/api => k8s.io/api v0.0.0-20190819141258-3544db3b9e44
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190819143637-0dbe462fe92d
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190819142446-92cc630367d0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190819144027-541433d7ce35
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190819141724-e14f31a72a77
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190819145148-d91c85d212d5
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190819145008-029dd04813af
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190819141909-f0f7c184477d
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190817025403-3ae76f584e79
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190819145328-4831a4ced492
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190819142756-13daafd3604f
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190819144832-f53437941eef
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190819144346-2e47de1df0f0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190819144657-d1a724e0828e
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20190602132728-7075c07e78bf
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190819144524-827174bad5e8
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190819145509-592c9a46fd00
	k8s.io/metrics => k8s.io/metrics v0.0.0-20190819143841-305e1cef1ab1
	k8s.io/node-api => k8s.io/node-api v0.0.0-20190819145652-b61681edbd0a
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190819143045-c84c31c165c4
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20190819144209-f9ca4b649af0
	k8s.io/sample-controller => k8s.io/sample-controller v0.0.0-20190819143301-7c475f5e1313
)

replace (
	github.com/networkservicemesh/networkservicemesh => ./
	github.com/networkservicemesh/networkservicemesh/controlplane => ./controlplane
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ./controlplane/api
	github.com/networkservicemesh/networkservicemesh/dataplane => ./dataplane
	github.com/networkservicemesh/networkservicemesh/dataplane/api => ./dataplane/api
	github.com/networkservicemesh/networkservicemesh/k8s => ./k8s
	github.com/networkservicemesh/networkservicemesh/k8s/api => ./k8s/api
	github.com/networkservicemesh/networkservicemesh/pkg => ./pkg
	github.com/networkservicemesh/networkservicemesh/sdk => ./sdk
	github.com/networkservicemesh/networkservicemesh/side-cars => ./side-cars
	github.com/networkservicemesh/networkservicemesh/test => ./test
	github.com/networkservicemesh/networkservicemesh/utils => ./utils
)

go 1.13
