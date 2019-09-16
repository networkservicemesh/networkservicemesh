module github.com/networkservicemesh/networkservicemesh/dataplane/api

require (
	github.com/golang/protobuf v1.3.2
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.1.0
	google.golang.org/grpc v1.23.0
)

replace github.com/networkservicemesh/networkservicemesh/controlplane/api => ./
