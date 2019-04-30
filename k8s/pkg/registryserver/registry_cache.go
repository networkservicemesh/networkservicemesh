package registryserver

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	nsmClientset "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resource_cache"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RegistryCache interface {
	AddNetworkService(ns *v1.NetworkService) (*v1.NetworkService, error)
	GetNetworkService(name string) (*v1.NetworkService, error)

	CreateOrUpdateNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error)
	GetNetworkServiceManager(name string) *v1.NetworkServiceManager

	AddNetworkServiceEndpoint(nse *v1.NetworkServiceEndpoint) (*v1.NetworkServiceEndpoint, error)
	DeleteNetworkServiceEndpoint(endpointName string) error
	GetEndpointsByNs(networkServiceName string) []*v1.NetworkServiceEndpoint
	GetEndpointsByNsm(nsmName string) []*v1.NetworkServiceEndpoint

	Start() error
	Stop()
}

type registryCacheImpl struct {
	networkServiceCache         *resource_cache.NetworkServiceCache
	networkServiceEndpointCache *resource_cache.NetworkServiceEndpointCache
	networkServiceManagerCache  *resource_cache.NetworkServiceManagerCache
	clientset                   *nsmClientset.Clientset
	stopFuncs                   []func()
	nsmNamespace                string
}

func NewRegistryCache(clientset *nsmClientset.Clientset) RegistryCache {
	return &registryCacheImpl{
		networkServiceCache:         resource_cache.NewNetworkServiceCache(),
		networkServiceEndpointCache: resource_cache.NewNetworkServiceEndpointCache(),
		networkServiceManagerCache:  resource_cache.NewNetworkServiceManagerCache(),
		clientset:                   clientset,
		stopFuncs:                   make([]func(), 0, 3),
		nsmNamespace:                namespace.GetNamespace(),
	}
}

func (rc *registryCacheImpl) Start() error {
	factory := externalversions.NewSharedInformerFactory(rc.clientset, 0)

	if stopFunc, err := rc.networkServiceCache.Start(factory); err != nil {
		rc.Stop()
		return err
	} else {
		rc.stopFuncs = append(rc.stopFuncs, stopFunc)
	}

	if stopFunc, err := rc.networkServiceEndpointCache.Start(factory); err != nil {
		rc.Stop()
		return err
	} else {
		rc.stopFuncs = append(rc.stopFuncs, stopFunc)
	}

	if stopFunc, err := rc.networkServiceManagerCache.Start(factory); err != nil {
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

	nsResponse, err := rc.clientset.NetworkservicemeshV1().NetworkServices(rc.nsmNamespace).Create(ns)
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
	if existingNse := rc.networkServiceEndpointCache.Get(nse.GetName()); existingNse != nil {
		return existingNse, nil
	}

	nseResponse, err := rc.clientset.NetworkservicemeshV1().NetworkServiceEndpoints(rc.nsmNamespace).Create(nse)
	if err == nil {
		rc.networkServiceEndpointCache.Add(nseResponse)
		return nseResponse, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nse, nil
	}

	return nil, err
}

func (rc *registryCacheImpl) DeleteNetworkServiceEndpoint(endpointName string) error {
	rc.networkServiceEndpointCache.Delete(endpointName)
	return rc.clientset.NetworkservicemeshV1().NetworkServiceEndpoints(rc.nsmNamespace).Delete(endpointName, &metav1.DeleteOptions{})
}

func (rc *registryCacheImpl) GetEndpointsByNs(networkServiceName string) []*v1.NetworkServiceEndpoint {
	return rc.networkServiceEndpointCache.GetByNetworkService(networkServiceName)
}

func (rc *registryCacheImpl) GetEndpointsByNsm(nsmName string) []*v1.NetworkServiceEndpoint {
	return rc.networkServiceEndpointCache.GetByNetworkServiceManager(nsmName)
}

func (rc *registryCacheImpl) CreateOrUpdateNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error) {
	existingNsm := rc.networkServiceManagerCache.Get(nsm.GetName())

	if existingNsm != nil {
		nsm.ObjectMeta = existingNsm.ObjectMeta
		logrus.Infof("NSM with name %v already exist in cache, updating: %v", nsm.GetName(), nsm)
		return rc.updateNetworkServiceManager(nsm)
	}

	logrus.Infof("Creating NSM: %v", nsm)
	createNsm, err := rc.addNetworkServiceManager(nsm)
	if err != nil || apierrors.IsAlreadyExists(err) {
		existingNsm, err = rc.clientset.NetworkservicemeshV1().NetworkServiceManagers(rc.nsmNamespace).Get(nsm.Name, metav1.GetOptions{})
		if err != nil {
			return existingNsm, err
		}
		nsm.ObjectMeta = existingNsm.ObjectMeta
		logrus.Infof("NSM with name %v already exist on server, updating cache: %v", nsm.GetName(), nsm)
		return rc.updateNetworkServiceManager(nsm)
	}
	return createNsm, err
}

func (rc *registryCacheImpl) addNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error) {
	nsmResponse, err := rc.clientset.NetworkservicemeshV1().NetworkServiceManagers(rc.nsmNamespace).Create(nsm)
	if err == nil {
		rc.networkServiceManagerCache.Add(nsmResponse)
	}
	return nsmResponse, err
}

func (rc *registryCacheImpl) updateNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error) {
	updNsm, err := rc.clientset.NetworkservicemeshV1().NetworkServiceManagers(rc.nsmNamespace).Update(nsm)
	if err == nil {
		rc.networkServiceManagerCache.Update(updNsm)
	}
	return updNsm, err
}

func (rc *registryCacheImpl) GetNetworkServiceManager(name string) *v1.NetworkServiceManager {
	return rc.networkServiceManagerCache.Get(name)
}

func (rc *registryCacheImpl) Stop() {
	for _, stopFunc := range rc.stopFuncs {
		stopFunc()
	}
}
