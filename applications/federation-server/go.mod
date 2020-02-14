module github.com/networkservicemesh/networkservicemesh/applications/federation-server

go 1.13

require (
	github.com/golang/protobuf v1.3.2
	github.com/networkservicemesh/networkservicemesh/sdk v0.3.0
	github.com/onsi/gomega v1.7.0
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spiffe/spire/proto/spire v0.9.2
	google.golang.org/grpc v1.27.0
)

replace (
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ../../controlplane/api
	github.com/networkservicemesh/networkservicemesh/pkg => ../../pkg
	github.com/networkservicemesh/networkservicemesh/sdk => ../../sdk
	github.com/networkservicemesh/networkservicemesh/utils => ../../utils
)
