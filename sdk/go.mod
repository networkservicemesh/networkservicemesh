module github.com/networkservicemesh/networkservicemesh/sdk

require (
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/protobuf v1.3.2
	github.com/hashicorp/go-multierror v1.0.0
	github.com/ligato/cn-infra v2.2.0+incompatible // indirect
	github.com/ligato/vpp-agent v2.1.1+incompatible
	github.com/mesos/mesos-go v0.0.9
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.2.0
	github.com/networkservicemesh/networkservicemesh/pkg v0.2.0
	github.com/networkservicemesh/networkservicemesh/utils v0.2.0
	github.com/onsi/gomega v1.7.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.8.1
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/teris-io/shortid v0.0.0-20171029131806-771a37caa5cf
	google.golang.org/grpc v1.23.1
)

replace (
	github.com/networkservicemesh/networkservicemesh => ../
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ../controlplane/api
	github.com/networkservicemesh/networkservicemesh/k8s/api => ../k8s/api
	github.com/networkservicemesh/networkservicemesh/pkg => ../pkg
	github.com/networkservicemesh/networkservicemesh/sdk => ./
	github.com/networkservicemesh/networkservicemesh/side-cars => ../side-cars
	github.com/networkservicemesh/networkservicemesh/utils => ../utils
)

go 1.13
