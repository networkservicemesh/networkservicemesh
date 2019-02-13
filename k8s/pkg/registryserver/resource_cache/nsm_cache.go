package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
)

type NetworkServiceManagerCache struct {
	cache                  abstractResourceCache
	networkServiceManagers map[string]*v1.NetworkServiceManager
}

func NewNetworkServiceManagerCache() *NetworkServiceManagerCache {
	rv := &NetworkServiceManagerCache{
		networkServiceManagers: make(map[string]*v1.NetworkServiceManager),
	}
	config := cacheConfig{
		keyFunc:             getNsmKey,
		resourceAddedFunc:   rv.resourceAdded,
		resourceDeletedFunc: rv.resourceDeleted,
		resourceType:        NsmResource,
	}
	rv.cache = newAbstractResourceCache(config)
	return rv
}

func (c *NetworkServiceManagerCache) Get(key string) *v1.NetworkServiceManager {
	return c.networkServiceManagers[key]
}

func (c *NetworkServiceManagerCache) Add(nsm *v1.NetworkServiceManager) {
	c.cache.add(nsm)
}

func (c *NetworkServiceManagerCache) Delete(key string) {
	c.cache.delete(key)
}

func (c *NetworkServiceManagerCache) Start(informerFactory externalversions.SharedInformerFactory) (func(), error) {
	return c.cache.start(informerFactory)
}

func (c *NetworkServiceManagerCache) resourceAdded(obj interface{}) {
	nsm := obj.(*v1.NetworkServiceManager)
	c.networkServiceManagers[getNsmKey(nsm)] = nsm
}

func (c *NetworkServiceManagerCache) resourceDeleted(key string) {
	delete(c.networkServiceManagers, key)
}

func getNsmKey(obj interface{}) string {
	return obj.(*v1.NetworkServiceManager).Name
}
