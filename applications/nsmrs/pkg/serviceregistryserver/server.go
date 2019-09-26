package serviceregistryserver

import (
	"net"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

// ServiceRegistry - service starting NSE registry server
type ServiceRegistry interface {
	NewPublicListener(registryAPIAddress string) (net.Listener, error)
}

type serviceRegistry struct {
}

// NewPublicListener - Starts public listener for NSMRS services
func (*serviceRegistry) NewPublicListener(registryAPIAddress string) (net.Listener, error) {
	return net.Listen("tcp", registryAPIAddress)
}

// NewNSMDServiceRegistryServer - creates new service registry service
func NewNSMDServiceRegistryServer() ServiceRegistry {
	return &serviceRegistry{}
}

// New - creates new grcp server and registers NSE discovery and registry services
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
