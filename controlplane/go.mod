module github.com/networkservicemesh/networkservicemesh/controlplane

require (
	github.com/golang/protobuf v1.3.2
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.2.0
	github.com/networkservicemesh/networkservicemesh/forwarder/api v0.2.0
	github.com/networkservicemesh/networkservicemesh/pkg v0.2.0
	github.com/networkservicemesh/networkservicemesh/sdk v0.2.0
	github.com/networkservicemesh/networkservicemesh/utils v0.2.0
	github.com/onsi/gomega v1.7.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/sys v0.0.0-20190618155005-516e3c20635f
	google.golang.org/grpc v1.23.1
)

replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8

replace (
	github.com/networkservicemesh/networkservicemesh => ../
	github.com/networkservicemesh/networkservicemesh/controlplane => ./
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ./api
	github.com/networkservicemesh/networkservicemesh/forwarder => ../forwarder
	github.com/networkservicemesh/networkservicemesh/forwarder/api => ../forwarder/api
	github.com/networkservicemesh/networkservicemesh/k8s/api => ../k8s/api
	github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis => ../k8s/pkg/apis
	github.com/networkservicemesh/networkservicemesh/pkg => ../pkg
	github.com/networkservicemesh/networkservicemesh/sdk => ../sdk
	github.com/networkservicemesh/networkservicemesh/side-cars => ../side-cars
	github.com/networkservicemesh/networkservicemesh/utils => ../utils
)

go 1.13
