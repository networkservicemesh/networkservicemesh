module github.com/networkservicemesh/networkservicemesh/controlplane/api

go 1.13

require (
	github.com/golang/protobuf v1.3.2
	github.com/networkservicemesh/networkservicemesh/utils v0.3.0
	github.com/pkg/errors v0.8.1
	google.golang.org/grpc v1.23.1
)

replace (
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ./
	github.com/networkservicemesh/networkservicemesh/utils => ../../utils
)
