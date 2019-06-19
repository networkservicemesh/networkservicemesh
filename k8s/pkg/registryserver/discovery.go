package registryserver

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
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
	st := time.Now()
	service, err := d.cache.GetNetworkService(request.NetworkServiceName)
	if err != nil {
		return nil, err
	}
	payload := service.Spec.Payload

	t1 := time.Now()
	endpointList := d.cache.GetEndpointsByNs(request.NetworkServiceName)
	logrus.Infof("NSE found %d, retrieve time: %v", len(endpointList), time.Since(t1))
	NSEs := make([]*registry.NetworkServiceEndpoint, len(endpointList))

	NSMs := make(map[string]*registry.NetworkServiceManager)
	endpointIds := []string{}
	for i, endpoint := range endpointList {
		NSEs[i] = mapNseFromCustomResource(endpoint)
		endpointIds = append(endpointIds, NSEs[i].EndpointName)
		nsm, err := d.cache.GetNetworkServiceManager(endpoint.Spec.NsmName)
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
