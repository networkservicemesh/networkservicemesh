module github.com/networkservicemesh/networkservicemesh/controlplane/api

require (
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/protobuf v1.3.2
	github.com/ligato/cn-infra v2.0.0+incompatible
	github.com/networkservicemesh/networkservicemesh v0.1.0
	github.com/sirupsen/logrus v1.4.2
	github.com/uber/jaeger-lib v2.1.1+incompatible // indirect
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/sys v0.0.0-20190618155005-516e3c20635f // indirect
	google.golang.org/grpc v1.23.0
)

replace (
	github.com/networkservicemesh/networkservicemesh/controlplane/api => ../controlplane/api
	github.com/networkservicemesh/networkservicemesh/k8s/api => ../k8s/api
)
