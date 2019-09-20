module github.com/networkservicemesh/networkservicemesh/dataplane

require (
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/protobuf v1.3.2
	github.com/ligato/cn-infra v2.0.0+incompatible
	github.com/ligato/vpp-agent v2.1.1+incompatible
	github.com/networkservicemesh/networkservicemesh/controlplane v0.2.0
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.2.0
	github.com/networkservicemesh/networkservicemesh/dataplane/api v0.2.0
	github.com/networkservicemesh/networkservicemesh/pkg v0.2.0
	github.com/networkservicemesh/networkservicemesh/sdk v0.2.0
	github.com/networkservicemesh/networkservicemesh/utils v0.2.0
	github.com/onsi/gomega v1.7.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/rs/xid v1.2.1
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20190625233234-7109fa855b0f
	google.golang.org/grpc v1.23.1
)

replace (
	github.com/networkservicemesh/networkservicemesh => ../
	github.com/networkservicemesh/networkservicemesh/controlplane => ../controlplane
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ../controlplane/api
	github.com/networkservicemesh/networkservicemesh/dataplane => ./
	github.com/networkservicemesh/networkservicemesh/dataplane/api => ./api
	github.com/networkservicemesh/networkservicemesh/k8s/api => ../k8s/api
	github.com/networkservicemesh/networkservicemesh/pkg => ../pkg
	github.com/networkservicemesh/networkservicemesh/sdk => ../sdk
	github.com/networkservicemesh/networkservicemesh/side-cars => ../side-cars
	github.com/networkservicemesh/networkservicemesh/utils => ../utils
)
