package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
)

type NetworkServiceCache struct {
	cache           abstractResourceCache
	networkServices map[string]*v1.NetworkService
}

func NewNetworkServiceCache() *NetworkServiceCache {
	rv := &NetworkServiceCache{
		networkServices: make(map[string]*v1.NetworkService),
	}
	config := cacheConfig{
		keyFunc:             getNsKey,
		resourceAddedFunc:   rv.resourceAdded,
		resourceDeletedFunc: rv.resourceDeleted,
		resourceType:        NsResource,
	}
	rv.cache = newAbstractResourceCache(config)
	return rv
}

func (c *NetworkServiceCache) Get(key string) *v1.NetworkService {
	return c.networkServices[key]
}

func (c *NetworkServiceCache) Add(ns *v1.NetworkService) {
	c.cache.add(ns)
}

func (c *NetworkServiceCache) Delete(key string) {
	c.cache.delete(key)
}

func (c *NetworkServiceCache) Start(informerFactory externalversions.SharedInformerFactory) (func(), error) {
	return c.cache.start(informerFactory)
}

func (c *NetworkServiceCache) resourceAdded(obj interface{}) {
	ns := obj.(*v1.NetworkService)
	c.networkServices[getNsKey(ns)] = ns
}

func (c *NetworkServiceCache) resourceDeleted(key string) {
	delete(c.networkServices, key)
}

func getNsKey(obj interface{}) string {
	return obj.(*v1.NetworkService).Name
}
