package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
)

type NetworkServiceCache struct {
	cache           abstractResourceCache
	networkServices map[string]*v1.NetworkService
	getCh           chan *v1.NetworkService
}

func NewNetworkServiceCache() *NetworkServiceCache {
	rv := &NetworkServiceCache{
		networkServices: make(map[string]*v1.NetworkService),
	}
	config := cacheConfig{
		keyFunc:             getNsKey,
		resourceAddedFunc:   rv.resourceAdded,
		resourceDeletedFunc: rv.resourceDeleted,
		resourceGetFunc:     rv.resourceGet,
		resourceType:        NsResource,
	}
	rv.cache = newAbstractResourceCache(config)
	return rv
}

func (c *NetworkServiceCache) Get(key string) *v1.NetworkService {
	v := c.cache.get(key)
	if v != nil {
		return v.(*v1.NetworkService)
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
	c.networkServices[ns.Name] = ns
}

func (c *NetworkServiceCache) resourceDeleted(key string) {
	delete(c.networkServices, key)
}

func (c *NetworkServiceCache) resourceGet(key string) interface{} {
	return c.networkServices[key]
}

func getNsKey(obj interface{}) string {
	return obj.(*v1.NetworkService).Name
}
