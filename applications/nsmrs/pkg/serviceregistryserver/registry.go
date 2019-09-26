package serviceregistryserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type nseRegistryService struct {
	cache NSERegistryCache
}

func NewNseRegistryService(cache NSERegistryCache) *nseRegistryService {
	return &nseRegistryService{
		cache: cache,
	}
}

func (rs *nseRegistryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("Received RegisterNSE(%v)", request)

	// Add public IP to NSM name to avoid name collision for different clusters
	nsmName := fmt.Sprintf("%s_%s", request.NetworkServiceManager.Name, request.NetworkServiceManager.Url)
	nsmName = strings.ReplaceAll(nsmName, ":", "_")
	request.NetworkServiceManager.Name = nsmName
	request.NetworkServiceEndpoint.NetworkServiceManagerName = nsmName

	monitor := NewNSMMonitor(request.NetworkServiceManager, func() {
		rs.RemoveNSE(ctx, &registry.RemoveNSERequest{
			NetworkServiceEndpointName: request.NetworkServiceEndpoint.Name,
		})
	})

	_, err := rs.cache.AddNetworkServiceEndpoint(&NSECacheEntry{
		nse:     request,
		monitor: monitor,
	})

	if err != nil {
		logrus.Errorf("Error registering NSE: %v", err)
		return nil, err
	}

	monitor.StartMonitor()

	logrus.Infof("Returned from RegisterNSE: request: %v", request)
	return request, err
}

func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	logrus.Infof("Received RemoveNSE(%v)", request)

	nse, err := rs.cache.DeleteNetworkServiceEndpoint(request.NetworkServiceEndpointName)
	nse.monitor.stop()
	if err != nil {
		logrus.Errorf("cannot remove Network Service Endpoint: %v", err)
		return &empty.Empty{}, err
	}

	logrus.Infof("RemoveNSE done: %v", nse)
	return &empty.Empty{}, nil
}
