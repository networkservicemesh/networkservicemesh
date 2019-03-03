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
	logrus.Info("RegistryCache started")

	rs := &registryService{
		nsmName: nsmName,
		cache:   cache,
	}

	ds := &discoveryService{
		nsmName: nsmName,
		cache:   cache,
	}

	registry.RegisterNetworkServiceRegistryServer(server, rs)
	registry.RegisterNetworkServiceDiscoveryServer(server, ds)

	if err := cache.Start(); err != nil {
		logrus.Error(err)
	}
	return server
}
