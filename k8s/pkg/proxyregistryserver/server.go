package proxyregistryserver

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/clusterinfo"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	nsmClientset "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver"
)

// New starts proxy Network Service Discovery Server and Cluster Info Server
func New(clientset *nsmClientset.Clientset, clusterInfoService clusterinfo.ClusterInfoServer) *grpc.Server {
	server := tools.NewServer(context.Background())
	cache := registryserver.NewRegistryCache(clientset, &registryserver.ResourceFilterConfig{})
	discovery := newDiscoveryService(cache, clusterInfoService)

	registry.RegisterNetworkServiceDiscoveryServer(server, discovery)
	clusterinfo.RegisterClusterInfoServer(server, clusterInfoService)

	if err := cache.Start(); err != nil {
		logrus.Error(err)
	}
	logrus.Info("RegistryCache started")

	return server
}
