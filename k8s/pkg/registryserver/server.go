package registryserver

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	nsmClientset "github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"google.golang.org/grpc"
)

func New(clientset *nsmClientset.Clientset, nsmName string) *grpc.Server {
	server := grpc.NewServer()

	srv := &registryService{
		clientset: clientset,
		nsmName:   nsmName,
	}
	registry.RegisterNetworkServiceRegistryServer(server, srv)
	registry.RegisterNetworkServiceDiscoveryServer(server, srv)
	return server
}
