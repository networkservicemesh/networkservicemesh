package registryserver

import (
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

const (
	// ForwardingTimeout - Timeout waiting for Proxy NseRegistryClient
	ForwardingTimeout = 15 * time.Second
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

	if request.GetNetworkServiceEndpoint() != nil && request.GetNetworkService() != nil {
		_, err := rs.cache.AddNetworkService(&v1.NetworkService{
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

		nseCr := mapNseToCustomResource(request.GetNetworkServiceEndpoint(), request.GetNetworkService(), rs.nsmName)
		if nseCr.GetName() == "" {
			nseCr.SetGenerateName(request.GetNetworkService().GetName())
		}

		nsePodName := tools.MetadataFromIncomingContext(ctx, endpoint.NSEPodNameMetadataKey)
		nsePodUID := tools.MetadataFromIncomingContext(ctx, endpoint.NSEPodUIDMetadataKey)
		if len(nsePodName) > 0 && len(nsePodUID) > 0 {
			nseCr.OwnerReferences = append(nseCr.OwnerReferences, generateOwnerReference(nsePodUID[0], nsePodName[0]))
		}

		nseResponse, err := rs.cache.AddNetworkServiceEndpoint(nseCr)
		if err != nil {
			nseCr.OwnerReferences = nil
			nseResponse, err = rs.cache.AddNetworkServiceEndpoint(nseCr)
			if err != nil {
				return nil, err
			}
		}

		request.NetworkServiceEndpoint = mapNseFromCustomResource(nseResponse)
		nsm, err := rs.cache.GetNetworkServiceManager(rs.nsmName)
		if err != nil {
			return nil, err
		}
		request.NetworkServiceManager = mapNsmFromCustomResource(nsm)
	}
	logrus.Infof("Returned from RegisterNSE: time: %v request: %v", time.Since(st), request)
	return request, nil
}

func (rs *nseRegistryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	st := time.Now()

	logrus.Infof("Received RemoveNSE(%v)", request)

	if err := rs.cache.DeleteNetworkServiceEndpoint(request.GetNetworkServiceEndpointName()); err != nil {
		return nil, err
	}
	logrus.Infof("RemoveNSE done: time %v", time.Since(st))
	return &empty.Empty{}, nil
}
