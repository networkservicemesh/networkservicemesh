package serviceregistryserver

import (
	"net"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type serviceRegistry struct {
}

func (*serviceRegistry) NewPublicListener(registryAPIAddress string) (net.Listener, error) {
	return net.Listen("tcp", registryAPIAddress)
}

func NewNSMDServiceRegistryServer() *serviceRegistry {
	return &serviceRegistry{}
}

func New() *grpc.Server {
	tracer := opentracing.GlobalTracer()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	cache := NewNSERegistryCache()
	discovery := newDiscoveryService(cache)
	registryService := NewNseRegistryService(cache)
	registry.RegisterNetworkServiceDiscoveryServer(server, discovery)
	registry.RegisterNetworkServiceRegistryServer(server, registryService)

	return server
}
