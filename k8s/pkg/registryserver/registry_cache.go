package registryserver

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	nsmClientset "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resource_cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type RegistryCache interface {
	AddNetworkService(ns *v1.NetworkService) (*v1.NetworkService, error)
	GetNetworkService(name string) (*v1.NetworkService, error)

	AddNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error)
	GetNetworkServiceManager(name string) (*v1.NetworkServiceManager, error)

	AddNetworkServiceEndpoint(nse *v1.NetworkServiceEndpoint) (*v1.NetworkServiceEndpoint, error)
	DeleteNetworkServiceEndpoint(endpointName string) error
	GetNetworkServiceEndpoints(networkServiceName string) []*v1.NetworkServiceEndpoint

	Start() error
	Stop()
}

type registryCacheImpl struct {
	networkServiceCache         *resource_cache.NetworkServiceCache
	networkServiceEndpointCache *resource_cache.NetworkServiceEndpointCache
	networkServiceManagerCache  *resource_cache.NetworkServiceManagerCache
	clientset                   *nsmClientset.Clientset
	stopFuncs                   []func()
}

func NewRegistryCache(clientset *nsmClientset.Clientset) RegistryCache {
	return &registryCacheImpl{
		networkServiceCache:         resource_cache.NewNetworkServiceCache(),
		networkServiceEndpointCache: resource_cache.NewNetworkServiceEndpointCache(),
		networkServiceManagerCache:  resource_cache.NewNetworkServiceManagerCache(),
		clientset:                   clientset,
		stopFuncs:                   make([]func(), 0, 3),
	}
}

func (rc *registryCacheImpl) Start() error {
	factory := externalversions.NewSharedInformerFactory(rc.clientset, 100*time.Millisecond)

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
	nsResponse, err := rc.clientset.NetworkservicemeshV1().NetworkServices("default").Create(ns)
	if nsResponse != nil {
		rc.networkServiceCache.Add(nsResponse)
	}
	return nsResponse, err
}

func (rc *registryCacheImpl) GetNetworkService(name string) (*v1.NetworkService, error) {
	if ns := rc.networkServiceCache.Get(name); ns == nil {
		return nil, fmt.Errorf("no NetworkService with name: %v", name)
	} else {
		return ns, nil
	}
}

func (rc *registryCacheImpl) AddNetworkServiceEndpoint(nse *v1.NetworkServiceEndpoint) (*v1.NetworkServiceEndpoint, error) {
	nseResponse, err := rc.clientset.NetworkservicemeshV1().NetworkServiceEndpoints("default").Create(nse)
	if nseResponse != nil {
		rc.networkServiceEndpointCache.Add(nseResponse)
	}
	return nseResponse, err
}

func (rc *registryCacheImpl) DeleteNetworkServiceEndpoint(endpointName string) error {
	rc.networkServiceEndpointCache.Delete(endpointName)
	return rc.clientset.NetworkservicemeshV1().NetworkServiceEndpoints("default").Delete(endpointName, &metav1.DeleteOptions{})
}

func (rc *registryCacheImpl) GetNetworkServiceEndpoints(networkServiceName string) []*v1.NetworkServiceEndpoint {
	return rc.networkServiceEndpointCache.Get(networkServiceName)
}

func (rc *registryCacheImpl) AddNetworkServiceManager(nsm *v1.NetworkServiceManager) (*v1.NetworkServiceManager, error) {
	nsmResponse, err := rc.clientset.NetworkservicemeshV1().NetworkServiceManagers("default").Create(nsm)
	if nsmResponse != nil {
		rc.networkServiceManagerCache.Add(nsmResponse)
	}
	return nsmResponse, err
}

func (rc *registryCacheImpl) GetNetworkServiceManager(name string) (*v1.NetworkServiceManager, error) {
	if nsm := rc.networkServiceManagerCache.Get(name); nsm == nil {
		return nil, fmt.Errorf("no NetworkServiceManager with name: %v", name)
	} else {
		return nsm, nil
	}
}

func (rc *registryCacheImpl) Stop() {
	for _, stopFunc := range rc.stopFuncs {
		stopFunc()
	}
}
