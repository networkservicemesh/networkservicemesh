package registryserver

import (
	"fmt"
	"time"

	"github.com/go-errors/errors"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NodeNameLabelKey = "nodeName"
)

type registryService struct {
	nsmName string
	cache   RegistryCache
}

type discoveryService struct {
	nsmName string
	cache   RegistryCache
}

func (rs *registryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	st := time.Now()

	logrus.Infof("Received RegisterNSE(%v)", request)
	// get network service
	if request.GetNetworkServiceManager().GetUrl() == "" {
		return nil, errors.New("NSERegistration.NetworkServiceManager.Url must be defined")
	}

	nsmCdr := mapNsmToCustomResource(request.NetworkServiceManager)
	nsmCdr.SetName(rs.nsmName)
	nsmCdr, err := rs.cache.AddNetworkServiceManager(nsmCdr)
	if err != nil {
		logrus.Errorf("Failed to register nsm: %s", err)
		return nil, err
	}

	request.NetworkServiceManager = mapNsmFromCustomResource(nsmCdr)

	labels := request.GetNetworkserviceEndpoint().GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["networkservicename"] = request.GetNetworkService().GetName()
	if request.GetNetworkserviceEndpoint() != nil && request.GetNetworkService() != nil {
		networkService, err := rs.cache.AddNetworkService(&v1.NetworkService{
			ObjectMeta: metav1.ObjectMeta{
				Name: request.NetworkService.GetName(),
			},
			Spec: v1.NetworkServiceSpec{
				Payload: request.NetworkService.GetPayload(),
			},
			Status: v1.NetworkServiceStatus{},
		})
		if err != nil {
			logrus.Errorf("Failed to register nsm: %s", err)
			return nil, err
		}
		nseResponse, err := rs.cache.AddNetworkServiceEndpoint(&v1.NetworkServiceEndpoint{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: request.GetNetworkService().GetName(),
				Labels:       labels,
			},
			Spec: v1.NetworkServiceEndpointSpec{
				NetworkServiceName: request.GetNetworkService().GetName(),
				NsmName:            rs.nsmName,
			},
			Status: v1.NetworkServiceEndpointStatus{
				State: v1.RUNNING,
			},
		})
		if err != nil {
			return nil, err
		}

		request.NetworkserviceEndpoint = &registry.NetworkServiceEndpoint{
			NetworkServiceName:        nseResponse.Spec.NetworkServiceName,
			Payload:                   networkService.Spec.Payload,
			NetworkServiceManagerName: nsmCdr.GetObjectMeta().GetName(),
			EndpointName:              nseResponse.GetObjectMeta().GetName(),
			Labels:                    nseResponse.GetObjectMeta().GetLabels(),
			State:                     string(nseResponse.Status.State),
		}
	}
	logrus.Infof("Returned from RegisterNSE: time: %v request: %v", time.Since(st), request)
	return request, nil

}

func (rs *registryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	st := time.Now()

	logrus.Infof("Received RemoveNSE(%v)", request)

	if err := rs.cache.DeleteNetworkServiceEndpoint(request.EndpointName); err != nil {
		return nil, err
	}
	logrus.Infof("RemoveNSE done: time %v", time.Since(st))
	return &empty.Empty{}, nil
}

func (rs *registryService) UpdateNSM(ctx context.Context, nsm *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	st := time.Now()

	logrus.Infof("Received UpdateNSM(%v)", nsm)

	if nsm.GetName() != rs.nsmName {
		return nil, fmt.Errorf("wrong nsm name %v, expected - %v", nsm.GetName(), rs.nsmName)
	}

	oldNsm, err := rs.cache.GetNetworkServiceManager(nsm.Name)
	if err != nil {
		return nil, err
	}

	nsmCdr := mapNsmToCustomResource(nsm)
	nsmCdr.ObjectMeta = oldNsm.ObjectMeta

	nsmCdr, err = rs.cache.UpdateNetworkServiceManager(nsmCdr)
	if err != nil {
		logrus.Errorf("Failed to register nsm: %s", err)
		return nil, err
	}

	logrus.Infof("UpdateNSE done: time %v", time.Since(st))
	return mapNsmFromCustomResource(nsmCdr), nil
}

func (rs *discoveryService) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	st := time.Now()
	service, err := rs.cache.GetNetworkService(request.NetworkServiceName)
	if err != nil {
		return nil, err
	}
	payload := service.Spec.Payload

	t1 := time.Now()
	endpointList := rs.cache.GetNetworkServiceEndpoints(request.NetworkServiceName)
	logrus.Infof("NSE found %d, retrieve time: %v", len(endpointList), time.Since(t1))
	NSEs := make([]*registry.NetworkServiceEndpoint, len(endpointList))

	NSMs := make(map[string]*registry.NetworkServiceManager)
	NSMsREG := make(map[string]*v1.NetworkServiceManager)
	for i, endpoint := range endpointList {
		NSEs[i] = &registry.NetworkServiceEndpoint{
			EndpointName:              endpoint.Name,
			NetworkServiceName:        endpoint.Spec.NetworkServiceName,
			NetworkServiceManagerName: endpoint.Spec.NsmName,
			Payload:                   payload,
			Labels:                    endpoint.ObjectMeta.Labels,
		}
		manager := NSMsREG[endpoint.Spec.NsmName]
		if manager == nil {
			manager, err = rs.cache.GetNetworkServiceManager(endpoint.Spec.NsmName)
			if err != nil {
				return nil, err
			}
			NSMsREG[endpoint.Spec.NsmName] = manager
		}
		NSMs[endpoint.Spec.NsmName] = mapNsmFromCustomResource(manager)
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
	logrus.Infof("FindNetworkService done: time %v", time.Since(st))
	return response, nil
}
