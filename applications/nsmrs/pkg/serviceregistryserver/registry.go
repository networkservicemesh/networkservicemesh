package serviceregistryserver

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/sirupsen/logrus"
)

type nseRegistryService struct {
	cache   NSERegistryCache
}

func NewNseRegistryService(cache NSERegistryCache) *nseRegistryService {
	return &nseRegistryService{
		cache: cache,
	}
}

func (rs *nseRegistryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("Received RegisterNSE(%v)", request)
	nse, err := rs.cache.AddNetworkServiceEndpoint(request)

	if err != nil {
		logrus.Errorf("Error registering NSE: %v", err)
		return nil, err
	}

	logrus.Infof("Returned from RegisterNSE: request: %v", request)
	return nse, err
}

func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	logrus.Infof("Received RemoveNSE(%v)", request)

	err := rs.cache.DeleteNetworkServiceEndpoint(request.NetworkServiceEndpointName)
	if err != nil {
		logrus.Errorf("cannot remove Network Service Endpoint: %v", err)
		return &empty.Empty{}, err
	}

	logrus.Infof("RemoveNSE done")
	return &empty.Empty{}, nil
}

