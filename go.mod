module github.com/networkservicemesh/networkservicemesh

require (
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/go-errors/errors v1.0.1
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/onsi/gomega v1.5.1-0.20190520121345-efe19c39ca10
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber/jaeger-client-go v2.16.0+incompatible
	github.com/uber/jaeger-lib v2.1.1+incompatible // indirect
	go.uber.org/atomic v1.4.0 // indirect
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc // indirect
	golang.org/x/sys v0.0.0-20190618155005-516e3c20635f // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/grpc v1.23.0
)

replace (
	github.com/networkservicemesh/networkservicemesh => ./
	github.com/networkservicemesh/networkservicemesh/controlplane => ./controlplane
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ./controlplane/api
	github.com/networkservicemesh/networkservicemesh/dataplane => ./dataplane
	github.com/networkservicemesh/networkservicemesh/dataplane/api => ./dataplane/api
	github.com/networkservicemesh/networkservicemesh/k8s/api => ./k8s/api
	github.com/networkservicemesh/networkservicemesh/sdk => ./sdk
	github.com/networkservicemesh/networkservicemesh/side-cars => ./side-cars
	github.com/networkservicemesh/networkservicemesh/test => ./test
)
