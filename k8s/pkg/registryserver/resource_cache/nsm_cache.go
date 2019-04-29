package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/sirupsen/logrus"
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
		resourceUpdatedFunc: rv.resourceUpdated,
		resourceGetFunc:     rv.resourceGet,
		resourceType:        NsmResource,
	}
	rv.cache = newAbstractResourceCache(config)
	return rv
}

func (c *NetworkServiceManagerCache) Get(key string) *v1.NetworkServiceManager {
	v := c.cache.get(key)
	if v != nil {
		return v.(*v1.NetworkServiceManager)
	}
	return nil
}

func (c *NetworkServiceManagerCache) Add(nsm *v1.NetworkServiceManager) {
	logrus.Infof("NetworkServiceManagerCache.Add(%v)", nsm)
	c.cache.add(nsm)
}

func (c *NetworkServiceManagerCache) Update(nsm *v1.NetworkServiceManager) {
	logrus.Infof("NetworkServiceManagerCache.Update(%v)", nsm)
	c.cache.update(nsm)
}

func (c *NetworkServiceManagerCache) Delete(key string) {
	logrus.Infof("NetworkServiceManagerCache.Delete(%v)", key)
	c.cache.delete(key)
}

func (c *NetworkServiceManagerCache) Start(informerFactory externalversions.SharedInformerFactory) (func(), error) {
	return c.cache.start(informerFactory)
}

func (c *NetworkServiceManagerCache) resourceAdded(obj interface{}) {
	nsm := obj.(*v1.NetworkServiceManager)
	logrus.Infof("NetworkServiceManagerCache.Added(%v)", nsm)
	c.networkServiceManagers[getNsmKey(nsm)] = nsm
}

func (c *NetworkServiceManagerCache) resourceUpdated(obj interface{}) {
	nsm := obj.(*v1.NetworkServiceManager)
	logrus.Infof("NetworkServiceManagerCache.resourceUpdated(%v)", nsm)
	c.networkServiceManagers[getNsmKey(nsm)] = nsm
}

func (c *NetworkServiceManagerCache) resourceDeleted(key string) {
	logrus.Infof("NetworkServiceManagerCache.Deleted(%v=%v)", key, c.networkServiceManagers[key])
	delete(c.networkServiceManagers, key)
}

func (c *NetworkServiceManagerCache) resourceGet(key string) interface{} {
	return c.networkServiceManagers[key]
}

func getNsmKey(obj interface{}) string {
	return obj.(*v1.NetworkServiceManager).Name
}
