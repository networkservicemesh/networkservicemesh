package registryserver

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resourcecache"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	nsmClientset "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
)

func New(ctx context.Context, clientset *nsmClientset.Clientset, nsmName string) *grpc.Server {
	span := spanhelper.FromContext(ctx, "K8SServer.New")
	defer span.Finish()
	server := tools.NewServer(span.Context())

	cache := NewRegistryCache(clientset, &ResourceFilterConfig{
		NetworkServiceManagerPolicy: resourcecache.FilterByNamespacePolicy(namespace.GetNamespace(), func(resource interface{}) string {
			nsm := resource.(*v1alpha1.NetworkServiceManager)
			return nsm.Namespace
		}),
	})

	nseRegistry := newNseRegistryService(nsmName, cache)
	nsmRegistry := newNsmRegistryService(nsmName, cache)
	discovery := newDiscoveryService(cache)

	registry.RegisterNetworkServiceRegistryServer(server, nseRegistry)
	registry.RegisterNetworkServiceDiscoveryServer(server, discovery)
	registry.RegisterNsmRegistryServer(server, nsmRegistry)

	if err := cache.Start(span.Context()); err != nil {
		logrus.Error(err)
	}
	logrus.Info("RegistryCache started")

	return server
}
