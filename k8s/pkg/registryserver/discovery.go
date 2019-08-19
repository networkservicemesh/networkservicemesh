package registryserver

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/utils"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
)

// Default values and environment variables of proxy connection
const (
	ProxyNsmdK8sAddressEnv      = "PROXY_NSMD_K8S_ADDRESS"
	ProxyNsmdK8sAddressDefaults = "pnsmgr-svc:5005"
)

type discoveryService struct {
	cache RegistryCache
}

func newDiscoveryService(cache RegistryCache) *discoveryService {
	return &discoveryService{
		cache: cache,
	}
}

func (d *discoveryService) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	if _, _, err := utils.ParseNsmURL(request.NetworkServiceName); err == nil {
		nsrURL := os.Getenv(ProxyNsmdK8sAddressEnv)
		if strings.TrimSpace(nsrURL) == "" {
			nsrURL = ProxyNsmdK8sAddressDefaults
		}
		remoteRegistry := nsmd.NewServiceRegistryAt(nsrURL)
		defer remoteRegistry.Stop()

		discoveryClient, err := remoteRegistry.DiscoveryClient()
		if err != nil {
			logrus.Error(err)
			return nil, err
		}

		logrus.Infof("Transfer request to proxy nsmd-k8s: %v", request)
		return discoveryClient.FindNetworkService(ctx, request)
	}

	return FindNetworkServiceWithCache(d.cache, request.NetworkServiceName)
}

// FindNetworkServiceWithCache returns network service with name from registry cache
func FindNetworkServiceWithCache(cache RegistryCache, networkServiceName string) (*registry.FindNetworkServiceResponse, error) {
	st := time.Now()
	service, err := cache.GetNetworkService(networkServiceName)
	if err != nil {
		return nil, err
	}
	payload := service.Spec.Payload

	t1 := time.Now()
	endpointList := cache.GetEndpointsByNs(networkServiceName)
	logrus.Infof("NSE found %d, retrieve time: %v", len(endpointList), time.Since(t1))
	NSEs := make([]*registry.NetworkServiceEndpoint, len(endpointList))

	NSMs := make(map[string]*registry.NetworkServiceManager)
	endpointIds := []string{}
	for i, endpoint := range endpointList {
		NSEs[i] = mapNseFromCustomResource(endpoint)
		endpointIds = append(endpointIds, NSEs[i].GetName())
		nsm, err := cache.GetNetworkServiceManager(endpoint.Spec.NsmName)
		if err != nil {
			return nil, err
		}
		NSMs[endpoint.Spec.NsmName] = mapNsmFromCustomResource(nsm)
	}

	var matches []*registry.Match

	for _, m := range service.Spec.Matches {
		var routes []*registry.Destination

		for _, r := range m.Routes {
			destination := &registry.Destination{
				DestinationSelector: r.DestinationSelector,
				Weight:              r.Weight,
			}
			routes = append(routes, destination)
		}

		match := &registry.Match{
			SourceSelector: m.SourceSelector,
			Routes:         routes,
		}
		matches = append(matches, match)
	}

	response := &registry.FindNetworkServiceResponse{
		Payload: payload,
		NetworkService: &registry.NetworkService{
			Name:    service.ObjectMeta.Name,
			Payload: service.Spec.Payload,
			Matches: matches,
		},
		NetworkServiceManagers:  NSMs,
		NetworkServiceEndpoints: NSEs,
	}

	logrus.Infof("FindNetworkService done: time %v %v", time.Since(st), endpointIds)
	return response, nil
}
