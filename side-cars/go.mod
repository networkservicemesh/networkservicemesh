module github.com/networkservicemesh/networkservicemesh/side-cars

require (
	github.com/networkservicemesh/networkservicemesh/controlplane v0.1.0
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.1.0
	github.com/networkservicemesh/networkservicemesh/k8s/api v0.1.0
	github.com/networkservicemesh/networkservicemesh/pkg v0.1.0
	github.com/networkservicemesh/networkservicemesh/sdk v0.1.0
	github.com/networkservicemesh/networkservicemesh/utils v0.1.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/sirupsen/logrus v1.4.2
)

replace (
	github.com/networkservicemesh/networkservicemesh => ../
	github.com/networkservicemesh/networkservicemesh/controlplane => ../controlplane
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ../controlplane/api
	github.com/networkservicemesh/networkservicemesh/dataplane => ../dataplane
	github.com/networkservicemesh/networkservicemesh/dataplane/api => ../dataplane/api
	github.com/networkservicemesh/networkservicemesh/k8s => ../k8s
	github.com/networkservicemesh/networkservicemesh/k8s/api => ../k8s/api
	github.com/networkservicemesh/networkservicemesh/pkg => ../pkg
	github.com/networkservicemesh/networkservicemesh/sdk => ../sdk
	github.com/networkservicemesh/networkservicemesh/side-cars => ../side-cars
	github.com/networkservicemesh/networkservicemesh/utils => ../utils
)
