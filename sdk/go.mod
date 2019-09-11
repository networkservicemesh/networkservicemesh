module github.com/networkservicemesh/networkservicemesh/sdk

require (
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.3.2
	github.com/hashicorp/go-multierror v1.0.0
	github.com/ligato/vpp-agent v2.1.1+incompatible
	github.com/mesos/mesos-go v0.0.9
	github.com/networkservicemesh/networkservicemesh v0.0.0-00010101000000-000000000000
	github.com/networkservicemesh/networkservicemesh/controlplane v0.0.0-00010101000000-000000000000
	github.com/onsi/gomega v1.5.1-0.20190520121345-efe19c39ca10
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/teris-io/shortid v0.0.0-20171029131806-771a37caa5cf
	google.golang.org/grpc v1.23.0
)

replace (
	github.com/networkservicemesh/networkservicemesh => ../
	github.com/networkservicemesh/networkservicemesh/controlplane => ../controlplane
	github.com/networkservicemesh/networkservicemesh/dataplane => ../dataplane
	github.com/networkservicemesh/networkservicemesh/sdk => ../sdk
)

go 1.13
