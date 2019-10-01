package registryserver

import (
	"context"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resourcecache"

	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	nsmClientset "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"
)

type RegistryCache interface {
	AddNetworkService(ns *v1.NetworkService) (*v1.NetworkService, error)
	GetNetworkService(name string) (*v1.NetworkService, error)

	CreateOrUpdateNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error)
	GetNetworkServiceManager(name string) (*v1.NetworkServiceManager, error)

	AddNetworkServiceEndpoint(nse *v1.NetworkServiceEndpoint) (*v1.NetworkServiceEndpoint, error)
	DeleteNetworkServiceEndpoint(endpointName string) error
	GetEndpointsByNs(networkServiceName string) []*v1.NetworkServiceEndpoint
	GetEndpointsByNsm(nsmName string) []*v1.NetworkServiceEndpoint

	Start(context.Context) error
	Stop()
}

type registryCacheImpl struct {
	networkServiceCache         *resourcecache.NetworkServiceCache
	networkServiceEndpointCache *resourcecache.NetworkServiceEndpointCache
	networkServiceManagerCache  *resourcecache.NetworkServiceManagerCache
	clientset                   *nsmClientset.Clientset
	stopFuncs                   []func()
	nsmNamespace                string
}

//ResourceFilterConfig means filter resource config for nsm custom resources
type ResourceFilterConfig struct {
	NetworkServiceEndpointFilterPolicy resourcecache.CacheFilterPolicy
	NetworkServiceManagerPolicy        resourcecache.CacheFilterPolicy
	NetworkServiceFilterPolicy         resourcecache.CacheFilterPolicy
}

func (conf *ResourceFilterConfig) setup() {
	if conf.NetworkServiceEndpointFilterPolicy == nil {
		conf.NetworkServiceEndpointFilterPolicy = resourcecache.NoFilterPolicy()
	}
	if conf.NetworkServiceManagerPolicy == nil {
		conf.NetworkServiceManagerPolicy = resourcecache.NoFilterPolicy()
	}
	if conf.NetworkServiceFilterPolicy == nil {
		conf.NetworkServiceFilterPolicy = resourcecache.NoFilterPolicy()
	}
}

//NewRegistryCache creates new registry cache
func NewRegistryCache(cs *nsmClientset.Clientset, conf *ResourceFilterConfig) RegistryCache {
	if conf == nil {
		conf = &ResourceFilterConfig{}
	}
	conf.setup()
	return &registryCacheImpl{
		networkServiceCache:         resourcecache.NewNetworkServiceCache(conf.NetworkServiceFilterPolicy),
		networkServiceEndpointCache: resourcecache.NewNetworkServiceEndpointCache(conf.NetworkServiceEndpointFilterPolicy),
		networkServiceManagerCache:  resourcecache.NewNetworkServiceManagerCache(conf.NetworkServiceManagerPolicy),
		clientset:                   cs,
		stopFuncs:                   make([]func(), 0, 3),
		nsmNamespace:                namespace.GetNamespace(),
	}
}

func (rc *registryCacheImpl) Start(ctx context.Context) error {
	span := spanhelper.FromContext(ctx, "RegistryCache.Start")
	defer span.Finish()
	factory := externalversions.NewSharedInformerFactory(rc.clientset, 0)

	if stopFunc, err := rc.networkServiceCache.StartWithResync(factory, rc.clientset); err != nil {
		rc.Stop()
		return err
	} else {
		rc.stopFuncs = append(rc.stopFuncs, stopFunc)
	}

	if stopFunc, err := rc.networkServiceEndpointCache.StartWithResync(factory, rc.clientset); err != nil {
		rc.Stop()
		return err
	} else {
		rc.stopFuncs = append(rc.stopFuncs, stopFunc)
	}

	if stopFunc, err := rc.networkServiceManagerCache.StartWithResync(factory, rc.clientset); err != nil {
		rc.Stop()
		return err
	} else {
		rc.stopFuncs = append(rc.stopFuncs, stopFunc)
	}

	return nil
}

func (rc *registryCacheImpl) AddNetworkService(ns *v1.NetworkService) (*v1.NetworkService, error) {
	if existingNs := rc.networkServiceCache.Get(ns.GetName()); existingNs != nil {
		return existingNs, nil
	}

	nsResponse, err := rc.clientset.NetworkservicemeshV1alpha1().NetworkServices(rc.nsmNamespace).Create(ns)
	if err == nil {
		rc.networkServiceCache.Add(nsResponse)
		return nsResponse, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return ns, nil
	}

	return nil, err
}

func (rc *registryCacheImpl) GetNetworkService(name string) (*v1.NetworkService, error) {
	if ns := rc.networkServiceCache.Get(name); ns == nil {
		return nil, fmt.Errorf("no NetworkService with name: %v", name)
	} else {
		return ns, nil
	}
}

func (rc *registryCacheImpl) AddNetworkServiceEndpoint(nse *v1.NetworkServiceEndpoint) (*v1.NetworkServiceEndpoint, error) {
	nseResponse, err := rc.clientset.NetworkservicemeshV1alpha1().NetworkServiceEndpoints(rc.nsmNamespace).Create(nse)
	if err == nil {
		rc.networkServiceEndpointCache.Add(nseResponse)
		return nseResponse, nil
	}

	return nil, err
}

func (rc *registryCacheImpl) DeleteNetworkServiceEndpoint(endpointName string) error {
	rc.networkServiceEndpointCache.Delete(endpointName)
	return rc.clientset.NetworkservicemeshV1alpha1().NetworkServiceEndpoints(rc.nsmNamespace).Delete(endpointName, &metav1.DeleteOptions{})
}

func (rc *registryCacheImpl) GetEndpointsByNs(networkServiceName string) []*v1.NetworkServiceEndpoint {
	return rc.networkServiceEndpointCache.GetByNetworkService(networkServiceName)
}

func (rc *registryCacheImpl) GetEndpointsByNsm(nsmName string) []*v1.NetworkServiceEndpoint {
	return rc.networkServiceEndpointCache.GetByNetworkServiceManager(nsmName)
}

const maxAllowedAttempts = 10

func (rc *registryCacheImpl) CreateOrUpdateNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error) {
	existingNsm := rc.networkServiceManagerCache.Get(nsm.GetName())

	attempt := 0
	for attempt < maxAllowedAttempts {
		logrus.Infof("CreateOrUpdateNSM attempt %d: ", attempt)

		if existingNsm == nil {
			logrus.Infof("Creating NSM: %v", nsm)
			newNsm, err := rc.addNetworkServiceManager(nsm)
			if err == nil || !apierrors.IsAlreadyExists(err) {
				return newNsm, err
			}

			logrus.Infof("NSM with name %v already exist", nsm.GetName())
			existingNsm, err = rc.GetNetworkServiceManager(nsm.GetName())
			if err != nil {
				return nil, err
			}
		} else {
			logrus.Infof("Updating existing NSM: %v with %v", existingNsm, nsm)
			updNsm := nsm.DeepCopy()
			updNsm.ObjectMeta = existingNsm.ObjectMeta
			updNsm, err := rc.updateNetworkServiceManager(updNsm)
			if err == nil || !apierrors.IsConflict(err) {
				return updNsm, err
			}

			logrus.Infof("There is no NSM with name %v", nsm.GetName())
			existingNsm = nil
		}
		attempt++
	}

	return nil, fmt.Errorf("exceeded the amount of attempts %d", maxAllowedAttempts)
}

func (rc *registryCacheImpl) addNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error) {
	nsmResponse, err := rc.clientset.NetworkservicemeshV1alpha1().NetworkServiceManagers(rc.nsmNamespace).Create(nsm)
	if err == nil {
		rc.networkServiceManagerCache.Add(nsmResponse)
	}
	return nsmResponse, err
}

func (rc *registryCacheImpl) updateNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error) {
	updNsm, err := rc.clientset.NetworkservicemeshV1alpha1().NetworkServiceManagers(rc.nsmNamespace).Update(nsm)
	if err == nil {
		rc.networkServiceManagerCache.Update(updNsm)
	}
	return updNsm, err
}

func (rc *registryCacheImpl) GetNetworkServiceManager(name string) (*v1.NetworkServiceManager, error) {
	if nsm := rc.networkServiceManagerCache.Get(name); nsm != nil {
		return nsm, nil
	}
	return rc.clientset.NetworkservicemeshV1alpha1().NetworkServiceManagers(rc.nsmNamespace).Get(name, metav1.GetOptions{})
}

func (rc *registryCacheImpl) Stop() {
	for _, stopFunc := range rc.stopFuncs {
		stopFunc()
	}
}
