package registryserver

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	nsmClientset "github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"golang.org/x/net/context"
)

type registryService struct{
	clientset *nsmClientset.Clientset
}

func (registryService) RegisterNSE(context.Context, *registry.RegisterNSERequest) (*registry.RegisterNSEResponse, error) {
	panic("implement me")
}

func (registryService) RemoveNSE(context.Context, *registry.RemoveNSERequest) (*registry.RemoveNSEResponse, error) {
	panic("implement me")
}

func (registryService) FindNetworkService(context.Context, *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	panic("implement me")
}
