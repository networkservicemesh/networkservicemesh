package registryserver

import (
	"time"

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

type nseRegistryService struct {
	nsmName string
	cache   RegistryCache
}

func newNseRegistryService(nsmName string, cache RegistryCache) *nseRegistryService {
	return &nseRegistryService{
		nsmName: nsmName,
		cache:   cache,
	}
}

func (rs *nseRegistryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	st := time.Now()

	logrus.Infof("Received RegisterNSE(%v)", request)

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

		request.NetworkserviceEndpoint = mapNseFromCustomResource(nseResponse, networkService.Spec.Payload)
		if nsm := rs.cache.GetNetworkServiceManager(rs.nsmName); nsm != nil {
			request.NetworkServiceManager = mapNsmFromCustomResource(nsm)
		}
	}
	logrus.Infof("Returned from RegisterNSE: time: %v request: %v", time.Since(st), request)
	return request, nil

}

func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	st := time.Now()

	logrus.Infof("Received RemoveNSE(%v)", request)

	if err := rs.cache.DeleteNetworkServiceEndpoint(request.EndpointName); err != nil {
		return nil, err
	}
	logrus.Infof("RemoveNSE done: time %v", time.Since(st))
	return &empty.Empty{}, nil
}
