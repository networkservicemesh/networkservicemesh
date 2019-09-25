module github.com/networkservicemesh/networkservicemesh/applications/nsmrs

go 1.13

require (
	github.com/golang/protobuf v1.3.2
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.2.0
	github.com/networkservicemesh/networkservicemesh/pkg v0.2.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/sirupsen/logrus v1.4.2
	google.golang.org/grpc v1.23.1
)

replace (
	github.com/networkservicemesh/networkservicemesh => ../../
	github.com/networkservicemesh/networkservicemesh/applications/nsmrs => ../..
	github.com/networkservicemesh/networkservicemesh/controlplane => ../../controlplane
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ../../controlplane/api
	github.com/networkservicemesh/networkservicemesh/dataplane/api => ../../dataplane/api
	github.com/networkservicemesh/networkservicemesh/k8s/api => ../../k8s/api
	github.com/networkservicemesh/networkservicemesh/pkg => ../../pkg
	github.com/networkservicemesh/networkservicemesh/sdk => ../../sdk
	github.com/networkservicemesh/networkservicemesh/side-cars => ../../side-cars
	github.com/networkservicemesh/networkservicemesh/utils => ../../utils
)
