package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"sync"
)

type NetworkServiceCache struct {
	cache           abstractResourceCache
	networkServices sync.Map
}

func NewNetworkServiceCache() *NetworkServiceCache {
	rv := &NetworkServiceCache{}
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
	if result, ok := c.networkServices.Load(key); ok {
		return result.(*v1.NetworkService)
	}
	return nil
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
	c.networkServices.Store(ns.Name, ns)
}

func (c *NetworkServiceCache) resourceDeleted(key string) {
	c.networkServices.Delete(key)
}

func getNsKey(obj interface{}) string {
	return obj.(*v1.NetworkService).Name
}
