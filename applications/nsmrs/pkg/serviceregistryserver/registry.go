package serviceregistryserver

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

// NSERegistryService - service registering Network Service Endpoints
type NSERegistryService interface {
	RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error)
	BulkRegisterNSE(registry.NetworkServiceRegistry_BulkRegisterNSEServer) error
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

// RegisterNSE - Registers NSE in cache
func (rs *nseRegistryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("Received RegisterNSE(%v)", request)

	request = prepareNSERequest(request)

	_, err := rs.cache.AddNetworkServiceEndpoint(request)
	if err != nil {
		logrus.Errorf("Error registering NSE: %v", err)
		return nil, err
	}

	logrus.Infof("Returned from RegisterNSE: request: %v", request)
	return request, err
}

func (rs *nseRegistryService) BulkRegisterNSE(srv registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	for {
		request, err := srv.Recv()
		if err != nil {
			err = fmt.Errorf("error receiving BulkRegisterNSE request : %v", err)
			return err
		}

		logrus.Infof("Received BulkRegisterNSE request: %v", request)

		request = prepareNSERequest(request)

		_, err = rs.cache.UpdateNetworkServiceEndpoint(request)
		if err != nil {
			err = fmt.Errorf("error processing BulkRegisterNSE request: %v", err)
			return err
		}
	}
	return nil
}

// RemoveNSE - Removes NSE from cache and stops NSMgr monitor
func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	logrus.Infof("Received RemoveNSE(%v)", request)

	nse, err := rs.cache.DeleteNetworkServiceEndpoint(request.NetworkServiceEndpointName)
	if err != nil {
		logrus.Errorf("cannot remove Network Service Endpoint: %v", err)
		return &empty.Empty{}, err
	}

	logrus.Infof("RemoveNSE done: %v", nse)
	return &empty.Empty{}, nil
}

func prepareNSERequest(request *registry.NSERegistration) *registry.NSERegistration {
	// Add public IP to NSM name to avoid name collision for different clusters
	nsmName := fmt.Sprintf("%s_%s", request.NetworkServiceManager.Name, request.NetworkServiceManager.Url)
	nsmName = strings.ReplaceAll(nsmName, ":", "_")
	request.NetworkServiceManager.Name = nsmName
	request.NetworkServiceEndpoint.NetworkServiceManagerName = nsmName

	return request
}