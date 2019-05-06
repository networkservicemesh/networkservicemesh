package registryserver

import (
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	nsmClientset "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func New(clientset *nsmClientset.Clientset, nsmName string) *grpc.Server {
	tracer := opentracing.GlobalTracer()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

	cache := NewRegistryCache(clientset)

	nseRegistry := newNseRegistryService(nsmName, cache)
	nsmRegistry := newNsmRegistryService(nsmName, cache)
	discovery := newDiscoveryService(cache)

	registry.RegisterNetworkServiceRegistryServer(server, nseRegistry)
	registry.RegisterNetworkServiceDiscoveryServer(server, discovery)
	registry.RegisterNsmRegistryServer(server, nsmRegistry)

	if err := cache.Start(); err != nil {
		logrus.Error(err)
	}
	logrus.Info("RegistryCache started")

	return server
}
