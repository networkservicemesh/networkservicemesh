package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

type NetworkServiceManagerCache struct {
	networkServiceManagers map[string]*v1.NetworkServiceManager
	addCh                  chan *v1.NetworkServiceManager
	deleteCh               chan *v1.NetworkServiceManager
}

func NewNetworkServiceManagerCache() *NetworkServiceManagerCache {
	return &NetworkServiceManagerCache{
		networkServiceManagers: make(map[string]*v1.NetworkServiceManager),
		addCh:                  make(chan *v1.NetworkServiceManager, 10),
		deleteCh:               make(chan *v1.NetworkServiceManager, 10),
	}
}

func (nsmCache *NetworkServiceManagerCache) Add(ns *v1.NetworkServiceManager) {
	nsmCache.addCh <- ns
}

func (nsmCache *NetworkServiceManagerCache) Get(name string) *v1.NetworkServiceManager {
	return nsmCache.networkServiceManagers[name]
}

func (nsmCache *NetworkServiceManagerCache) Delete(ns *v1.NetworkServiceManager) {
	nsmCache.deleteCh <- ns
}

func (nsmCache *NetworkServiceManagerCache) GetResourceType() string {
	return NsmResource
}

func (nsmCache *NetworkServiceManagerCache) Run(informerFactory externalversions.SharedInformerFactory) error {
	if err := nsmCache.startInformer(informerFactory); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case newNsm := <-nsmCache.addCh:
				nsmCache.networkServiceManagers[newNsm.Name] = newNsm
			case deleteNsm := <-nsmCache.deleteCh:
				delete(nsmCache.networkServiceManagers, deleteNsm.Name)
			}
		}
	}()

	return nil
}

func (nsCache *NetworkServiceManagerCache) startInformer(informerFactory externalversions.SharedInformerFactory) error {
	genericInformer, err := informerFactory.ForResource(v1.SchemeGroupVersion.WithResource(NsmResource))
	if err != nil {
		return err
	}
	informer := genericInformer.Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { nsCache.Add(obj.(*v1.NetworkServiceManager)) },
		DeleteFunc: func(obj interface{}) { nsCache.Delete(obj.(*v1.NetworkServiceManager)) },
	})
	stopper := make(chan struct{})
	go informer.Run(stopper)
	return nil
}
