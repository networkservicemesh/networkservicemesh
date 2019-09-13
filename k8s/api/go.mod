module github.com/networkservicemesh/networkservicemesh/k8s/api

require (
	github.com/golang/protobuf v1.3.2
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.0.0-00010101000000-000000000000
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20190611190212-a7e196e89fd3 // indirect
	google.golang.org/grpc v1.23.0
)

replace github.com/networkservicemesh/networkservicemesh/controlplane/api => ../../controlplane/api
