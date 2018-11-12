package registryserver

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	nsmClientset "github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"google.golang.org/grpc"
)

func New(clientset *nsmClientset.Clientset) *grpc.Server {
	server := grpc.NewServer()
	registry.RegisterNetworkServiceRegistryServer(server, &registryService{
		clientset: clientset,
	})
	return server
}
