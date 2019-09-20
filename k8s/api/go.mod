module github.com/networkservicemesh/networkservicemesh/k8s/api

require (
	github.com/golang/protobuf v1.3.2
	github.com/networkservicemesh/networkservicemesh/controlplane/api v0.2.0
	google.golang.org/appengine v1.4.0 // indirect
	google.golang.org/genproto v0.0.0-20190611190212-a7e196e89fd3 // indirect
	google.golang.org/grpc v1.23.1
)

replace github.com/networkservicemesh/networkservicemesh/controlplane/api => ../../controlplane/api

replace github.com/networkservicemesh/networkservicemesh/pkg => ../../pkg

replace github.com/networkservicemesh/networkservicemesh/utils => ../../utils
