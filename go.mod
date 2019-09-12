module github.com/networkservicemesh/networkservicemesh

require (
	github.com/go-errors/errors v1.0.1
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/networkservicemesh/networkservicemesh/controlplane v0.0.0-00010101000000-000000000000
	github.com/onsi/gomega v1.5.1-0.20190520121345-efe19c39ca10
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/uber/jaeger-client-go v2.16.0+incompatible
	google.golang.org/grpc v1.23.0
)

replace (
	github.com/networkservicemesh/networkservicemesh => ./
	github.com/networkservicemesh/networkservicemesh/controlplane => ./controlplane
	github.com/networkservicemesh/networkservicemesh/dataplane => ./dataplane
	github.com/networkservicemesh/networkservicemesh/sdk => ./sdk
	github.com/networkservicemesh/networkservicemesh/side-cars => ./side-cars
	github.com/networkservicemesh/networkservicemesh/test => ./test
)
