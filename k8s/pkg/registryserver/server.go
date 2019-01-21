package registryserver

import (
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	nsmClientset "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

func New(clientset *nsmClientset.Clientset, nsmName string) *grpc.Server {
	tracer := opentracing.GlobalTracer()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	srv := &registryService{
		clientset: clientset,
		nsmName:   nsmName,
	}
	registry.RegisterNetworkServiceRegistryServer(server, srv)
	registry.RegisterNetworkServiceDiscoveryServer(server, srv)
	return server
}
