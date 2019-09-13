module github.com/networkservicemesh/networkservicemesh/dataplane/api

require (
	github.com/golang/protobuf v1.3.2
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.23.0
)

replace github.com/networkservicemesh/networkservicemesh/controlplane/api => ../../controlplane/api
