module github.com/networkservicemesh/networkservicemesh/utils

require (
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.2.0
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.7.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber/jaeger-lib v2.1.1+incompatible // indirect
	github.com/vishvananda/netns v0.0.0-20190625233234-7109fa855b0f
	go.uber.org/atomic v1.4.0 // indirect
)

replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8

replace (
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ../controlplane/api
	github.com/networkservicemesh/networkservicemesh/pkg => ../pkg
	github.com/networkservicemesh/networkservicemesh/utils => ./
)

go 1.13
