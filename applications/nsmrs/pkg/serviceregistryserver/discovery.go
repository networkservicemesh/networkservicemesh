package serviceregistryserver

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type discoveryService struct {

}

func newDiscoveryService() *discoveryService {
	return &discoveryService{

	}
}

func (d *discoveryService) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {

	return nil, fmt.Errorf("not implemented")
}
