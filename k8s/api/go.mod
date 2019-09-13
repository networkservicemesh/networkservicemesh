module github.com/networkservicemesh/networkservicemesh/k8s/api

require (
	github.com/golang/protobuf v1.3.2
	github.com/ligato/cn-infra v2.0.0+incompatible
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.1.0
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/genproto v0.0.0-20190611190212-a7e196e89fd3 // indirect
	google.golang.org/grpc v1.23.0
)

replace github.com/networkservicemesh/networkservicemesh/controlplane/api => ../../controlplane/api
