package serviceregistryserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

// NSERegistryService - service registering Network Service Endpoints
type NSERegistryService interface {
	RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error)
	RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error)
}

type nseRegistryService struct {
	cache NSERegistryCache
}

// NewNseRegistryService - creates NSE Registry service
func NewNseRegistryService(cache NSERegistryCache) NSERegistryService {
	return &nseRegistryService{
		cache: cache,
	}
}

// RegisterNSE - Registers NSE in cache and starts NSMgr monitor
func (rs *nseRegistryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("Received RegisterNSE(%v)", request)

	// Add public IP to NSM name to avoid name collision for different clusters
	nsmName := fmt.Sprintf("%s_%s", request.NetworkServiceManager.Name, request.NetworkServiceManager.Url)
	nsmName = strings.ReplaceAll(nsmName, ":", "_")
	request.NetworkServiceManager.Name = nsmName
	request.NetworkServiceEndpoint.NetworkServiceManagerName = nsmName

	monitor := NewNSMMonitor(request.NetworkServiceManager, func() {
		_, err := rs.RemoveNSE(ctx, &registry.RemoveNSERequest{
			NetworkServiceEndpointName: request.NetworkServiceEndpoint.Name,
		})
		if err != nil {
			logrus.Errorf("Error removing Network Service Endpoint (%s) from cache: %v", request.NetworkServiceEndpoint.Name, err)
		}
	})

	_, err := rs.cache.AddNetworkServiceEndpoint(&NSECacheEntry{
		nse:     request,
		monitor: monitor,
	})

	if err != nil {
		logrus.Errorf("Error registering NSE: %v", err)
		return nil, err
	}

	err = monitor.StartMonitor()
	if err != nil {
		logrus.Errorf("Error starting NSMgr monitor: %v", err)
		_, removeErr := rs.RemoveNSE(ctx, &registry.RemoveNSERequest{
			NetworkServiceEndpointName: request.NetworkServiceEndpoint.Name,
		})
		if removeErr != nil {
			logrus.Errorf("Error removing Network Service Endpoint (%s) from cache: %v", request.NetworkServiceEndpoint.Name, removeErr)
		}
	}

	logrus.Infof("Returned from RegisterNSE: request: %v", request)
	return request, err
}

// RemoveNSE - Removes NSE from cache and stops NSMgr monitor
func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	logrus.Infof("Received RemoveNSE(%v)", request)

	nse, err := rs.cache.DeleteNetworkServiceEndpoint(request.NetworkServiceEndpointName)
	nse.monitor.Stop()
	if err != nil {
		logrus.Errorf("cannot remove Network Service Endpoint: %v", err)
		return &empty.Empty{}, err
	}

	logrus.Infof("RemoveNSE done: %v", nse)
	return &empty.Empty{}, nil
}
