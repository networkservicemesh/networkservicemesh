package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

type NetworkServiceCache struct {
	networkServices map[string]*v1.NetworkService
	addCh           chan *v1.NetworkService
	deleteCh        chan *v1.NetworkService
}

func NewNetworkServiceCache() *NetworkServiceCache {
	return &NetworkServiceCache{
		networkServices: make(map[string]*v1.NetworkService),
		addCh:           make(chan *v1.NetworkService, 10),
		deleteCh:        make(chan *v1.NetworkService, 10),
	}
}

func (nsCache *NetworkServiceCache) Add(ns *v1.NetworkService) {
	nsCache.addCh <- ns
}

func (nsCache *NetworkServiceCache) Get(name string) *v1.NetworkService {
	return nsCache.networkServices[name]
}

func (nsCache *NetworkServiceCache) Delete(ns *v1.NetworkService) {
	nsCache.deleteCh <- ns
}

func (nsCache *NetworkServiceCache) GetResourceType() string {
	return NsResource
}

func (nsCache *NetworkServiceCache) Run(informerFactory externalversions.SharedInformerFactory) error {
	if err := nsCache.startInformer(informerFactory); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case newNetworkService := <-nsCache.addCh:
				nsCache.networkServices[newNetworkService.Name] = newNetworkService
			case deleteNetworkService := <-nsCache.deleteCh:
				delete(nsCache.networkServices, deleteNetworkService.Name)
			}
		}
	}()

	return nil
}

func (nsCache *NetworkServiceCache) startInformer(informerFactory externalversions.SharedInformerFactory) error {
	genericInformer, err := informerFactory.ForResource(v1.SchemeGroupVersion.WithResource(NsResource))
	if err != nil {
		return err
	}
	informer := genericInformer.Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { nsCache.Add(obj.(*v1.NetworkService)) },
		DeleteFunc: func(obj interface{}) { nsCache.Delete(obj.(*v1.NetworkService)) },
	})
	stopper := make(chan struct{})
	go informer.Run(stopper)
	return nil
}
