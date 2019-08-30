package registryserver

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resourcecache"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	nsmClientset "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
)

func New(clientset *nsmClientset.Clientset, nsmName string) *grpc.Server {
	server := tools.NewServer()

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

	if err := cache.Start(); err != nil {
		logrus.Error(err)
	}
	logrus.Info("RegistryCache started")

	return server
}
