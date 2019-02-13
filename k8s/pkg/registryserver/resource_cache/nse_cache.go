package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
)

type NetworkServiceEndpointCache struct {
	nseByNs                 map[string][]*v1.NetworkServiceEndpoint
	networkServiceEndpoints map[string]*v1.NetworkServiceEndpoint
	addCh                   chan *v1.NetworkServiceEndpoint
	deleteCh                chan string
}

func NewNetworkServiceEndpointCache() *NetworkServiceEndpointCache {
	return &NetworkServiceEndpointCache{
		nseByNs:                 make(map[string][]*v1.NetworkServiceEndpoint),
		networkServiceEndpoints: make(map[string]*v1.NetworkServiceEndpoint),
		addCh:                   make(chan *v1.NetworkServiceEndpoint, 10),
		deleteCh:                make(chan string, 10),
	}
}

func (nseCache *NetworkServiceEndpointCache) Add(nse *v1.NetworkServiceEndpoint) {
	logrus.Infof("Adding NSE to cache: %v", *nse)
	nseCache.addCh <- nse
}

func (nseCache *NetworkServiceEndpointCache) Get(networkServiceName string) []*v1.NetworkServiceEndpoint {
	return nseCache.nseByNs[networkServiceName]
}

func (nseCache *NetworkServiceEndpointCache) Delete(name string) {
	logrus.Infof("Deleting NSE from cache: %v", name)
	nseCache.deleteCh <- name
}

func (nseCache *NetworkServiceEndpointCache) GetResourceType() string {
	return NseResource
}

func (nseCache *NetworkServiceEndpointCache) Run(informerFactory externalversions.SharedInformerFactory) error {
	if err := nseCache.startInformer(informerFactory); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case newNse := <-nseCache.addCh:
				endpoints := nseCache.nseByNs[newNse.Spec.NetworkServiceName]
				if _, exist := nseCache.networkServiceEndpoints[newNse.Name]; !exist {
					nseCache.nseByNs[newNse.Spec.NetworkServiceName] = append(endpoints, newNse)
				} else {
					for i, e := range endpoints {
						if e.Name == newNse.Name {
							endpoints[i] = newNse
							break
						}
					}
				}
				nseCache.networkServiceEndpoints[newNse.Name] = newNse
			case deleteNseName := <-nseCache.deleteCh:
				nse, exist := nseCache.networkServiceEndpoints[deleteNseName]
				if !exist {
					continue
				}

				endpoints := nseCache.nseByNs[nse.Spec.NetworkServiceName]
				logrus.Info("endpoints: %v", endpoints)
				var index int
				for i, e := range endpoints {
					if nse.Name == e.Name {
						index = i
						break
					}
				}
				endpoints = append(endpoints[:index], endpoints[index+1:]...)
				if len(endpoints) == 0 {
					delete(nseCache.nseByNs, nse.Spec.NetworkServiceName)
				} else {
					nseCache.nseByNs[nse.Spec.NetworkServiceName] = endpoints
				}
				delete(nseCache.networkServiceEndpoints, deleteNseName)
			}
		}
	}()

	return nil
}

func (nseCache *NetworkServiceEndpointCache) startInformer(informerFactory externalversions.SharedInformerFactory) error {
	genericInformer, err := informerFactory.ForResource(v1.SchemeGroupVersion.WithResource(NseResource))
	if err != nil {
		return err
	}
	informer := genericInformer.Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { nseCache.Add(obj.(*v1.NetworkServiceEndpoint)) },
		DeleteFunc: func(obj interface{}) { nseCache.Delete(obj.(*v1.NetworkServiceEndpoint).Name) },
	})
	stopper := make(chan struct{})
	go informer.Run(stopper)
	return nil
}
