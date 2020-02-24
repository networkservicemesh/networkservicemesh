module github.com/networkservicemesh/networkservicemesh/controlplane

go 1.13

require (
	github.com/golang/protobuf v1.3.3
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.3.0
	github.com/networkservicemesh/networkservicemesh/forwarder/api v0.3.0
	github.com/networkservicemesh/networkservicemesh/pkg v0.3.0
	github.com/networkservicemesh/networkservicemesh/sdk v0.3.0
	github.com/networkservicemesh/networkservicemesh/utils v0.3.0
	github.com/onsi/gomega v1.7.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v0.9.3
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa
	golang.org/x/sys v0.0.0-20200124204421-9fbb57f87de9
	google.golang.org/grpc v1.27.0
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
