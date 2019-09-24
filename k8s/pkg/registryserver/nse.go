package registryserver

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	logrus.Infof("Received RegisterNSE(%v)", request)

	labels := request.GetNetworkServiceEndpoint().GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["networkservicename"] = request.GetNetworkService().GetName()
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

		var objectMeta metav1.ObjectMeta
		if request.GetNetworkServiceEndpoint().GetName() == "" {
			objectMeta = metav1.ObjectMeta{
				GenerateName: request.GetNetworkService().GetName(),
				Labels:       labels,
			}
		} else {
			objectMeta = metav1.ObjectMeta{
				Name:   request.GetNetworkServiceEndpoint().GetName(),
				Labels: labels,
			}
		}

		nseResponse, err := rs.cache.AddNetworkServiceEndpoint(&v1.NetworkServiceEndpoint{
			ObjectMeta: objectMeta,
			Spec: v1.NetworkServiceEndpointSpec{
				NetworkServiceName: request.GetNetworkService().GetName(),
				Payload:            request.GetNetworkService().GetPayload(),
				NsmName:            rs.nsmName,
			},
			Status: v1.NetworkServiceEndpointStatus{
				State: v1.RUNNING,
			},
		})
		if err != nil {
			return nil, err
		}

		request.NetworkServiceEndpoint = mapNseFromCustomResource(nseResponse)
		nsm, err := rs.cache.GetNetworkServiceManager(rs.nsmName)
		if err != nil {
			return nil, err
		}
		request.NetworkServiceManager = mapNsmFromCustomResource(nsm)

		err = rs.forwardRegisterNSE(ctx, request)
		if err != nil {
			logrus.Errorf("Cannot forward NSE Registration: %v", err)
		}
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

	if err := rs.forwardRemoveNSE(ctx, request); err != nil {
		logrus.Errorf("Cannot forward Remove NSE: %v", err)
	}

	logrus.Infof("RemoveNSE done: time %v", time.Since(st))
	return &empty.Empty{}, nil
}

func (rs *nseRegistryService) forwardRegisterNSE(ctx context.Context, request *registry.NSERegistration) error {
	nsrURL := os.Getenv(ProxyNsmdK8sAddressEnv)
	if strings.TrimSpace(nsrURL) == "" {
		nsrURL = ProxyNsmdK8sAddressDefaults
	}

	remoteRegistry := nsmd.NewServiceRegistryAt(nsrURL)
	defer remoteRegistry.Stop()

	nseRegistryClient, err := remoteRegistry.NseRegistryClient()
	if err != nil {
		return err
	}

	service, err := rs.cache.GetNetworkService(request.NetworkService.Name)
	if err != nil {
		return err
	}

	request.NetworkService.Payload = service.Spec.Payload

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
		request.NetworkService.Matches = append(request.NetworkService.Matches, match)
	}

	logrus.Info("Forwarding Register NSE request (%v)", request)
	_, err = nseRegistryClient.RegisterNSE(ctx, request)
	if err != nil {
		return err
	}

	return nil
}

func (rs *nseRegistryService) forwardRemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) error {
	nsrURL := os.Getenv(ProxyNsmdK8sAddressEnv)
	if strings.TrimSpace(nsrURL) == "" {
		nsrURL = ProxyNsmdK8sAddressDefaults
	}

	remoteRegistry := nsmd.NewServiceRegistryAt(nsrURL)
	defer remoteRegistry.Stop()

	nseRegistryClient, err := remoteRegistry.NseRegistryClient()
	if err != nil {
		return err
	}

	logrus.Info("Forwarding Remove NSE request (%v)", request)
	_, err = nseRegistryClient.RemoveNSE(ctx, request)
	if err != nil {
		return err
	}

	return nil
}