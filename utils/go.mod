module github.com/networkservicemesh/networkservicemesh

require (
	github.com/go-errors/errors v1.0.1
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.1.0
	github.com/networkservicemesh/networkservicemesh/utils v0.1.0
	github.com/onsi/gomega v1.5.1-0.20190520121345-efe19c39ca10
	github.com/sirupsen/logrus v1.4.2
)

replace (
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ../controlplane/api
	github.com/networkservicemesh/networkservicemesh/utils => ./
)
