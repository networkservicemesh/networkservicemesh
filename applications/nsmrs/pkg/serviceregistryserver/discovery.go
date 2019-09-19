package serviceregistryserver

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/sirupsen/logrus"
)

type discoveryService struct {

}

func newDiscoveryService() *discoveryService {
	return &discoveryService{

	}
}

func (d *discoveryService) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	logrus.Errorf("Not implemented")
	return nil, fmt.Errorf("not implemented")
}
