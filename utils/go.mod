module github.com/networkservicemesh/networkservicemesh/utils

require (
	github.com/go-errors/errors v1.0.1
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.2.0
        github.com/networkservicemesh/networkservicemesh/utils v0.2.0
	github.com/onsi/gomega v1.7.0
	github.com/sirupsen/logrus v1.4.2
)

replace (
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ../controlplane/api
	github.com/networkservicemesh/networkservicemesh/pkg => ../pkg
	github.com/networkservicemesh/networkservicemesh/utils => ./
)

go 1.13
