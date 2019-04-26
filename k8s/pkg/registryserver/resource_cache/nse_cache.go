package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/sirupsen/logrus"
	"sync"
)

type NetworkServiceEndpointCache struct {
	cache                   abstractResourceCache
	nseByNs                 sync.Map
	networkServiceEndpoints sync.Map
}

func NewNetworkServiceEndpointCache() *NetworkServiceEndpointCache {
	rv := &NetworkServiceEndpointCache{}
	config := cacheConfig{
		keyFunc:             getNseKey,
		resourceAddedFunc:   rv.resourceAdded,
		resourceDeletedFunc: rv.resourceDeleted,
		resourceType:        NseResource,
	}
	rv.cache = newAbstractResourceCache(config)
	return rv
}

func (c *NetworkServiceEndpointCache) Get(key string) *v1.NetworkServiceEndpoint {
	if result, ok := c.networkServiceEndpoints.Load(key); ok {
		return result.(*v1.NetworkServiceEndpoint)
	}
	return nil
}

func (c *NetworkServiceEndpointCache) GetByNetworkService(networkServiceName string) []*v1.NetworkServiceEndpoint {
	if result, ok := c.nseByNs.Load(networkServiceName); ok {
		return result.([]*v1.NetworkServiceEndpoint)
	}
	return nil
}

func (c *NetworkServiceEndpointCache) GetByNetworkServiceManager(nsmName string) []*v1.NetworkServiceEndpoint {
	var rv []*v1.NetworkServiceEndpoint

	c.networkServiceEndpoints.Range(func(key, value interface{}) bool {
		endpoint := value.(*v1.NetworkServiceEndpoint)
		if endpoint.Spec.NsmName == nsmName {
			rv = append(rv, endpoint)
		}
		return true
	})

	return rv
}

func (c *NetworkServiceEndpointCache) Add(nse *v1.NetworkServiceEndpoint) {
	logrus.Infof("Adding NSE to cache: %v", *nse)
	c.cache.add(nse)
}

func (c *NetworkServiceEndpointCache) Delete(key string) {
	c.cache.delete(key)
}

func (c *NetworkServiceEndpointCache) Start(informerFactory externalversions.SharedInformerFactory) (func(), error) {
	return c.cache.start(informerFactory)
}

func (c *NetworkServiceEndpointCache) resourceAdded(obj interface{}) {
	nse := obj.(*v1.NetworkServiceEndpoint)
	var endpoints []*v1.NetworkServiceEndpoint
	if val, ok := c.nseByNs.Load(nse.Spec.NetworkServiceName); ok {
		endpoints = val.([]*v1.NetworkServiceEndpoint)
	} else {
		endpoints = []*v1.NetworkServiceEndpoint{}
	}
	if _, exist := c.networkServiceEndpoints.Load(getNseKey(nse)); !exist {
		c.nseByNs.Store(nse.Spec.NetworkServiceName, append(endpoints, nse))
	} else {
		for i, e := range endpoints {
			if getNseKey(nse) == getNseKey(e) {
				endpoints[i] = nse
				break
			}
		}
	}
	c.networkServiceEndpoints.Store(getNseKey(nse), nse)
}

func (c *NetworkServiceEndpointCache) resourceDeleted(key string) {
	val, exist := c.networkServiceEndpoints.Load(key)
	if !exist {
		return
	}
	nse := val.(*v1.NetworkServiceEndpoint)
	var endpoints []*v1.NetworkServiceEndpoint
	if val, ok := c.nseByNs.Load(nse.Spec.NetworkServiceName); ok {
		endpoints = val.([]*v1.NetworkServiceEndpoint)
	} else {
		endpoints = []*v1.NetworkServiceEndpoint{}
	}
	var index int
	for i, e := range endpoints {
		if getNseKey(nse) == getNseKey(e) {
			index = i
			break
		}
	}
	endpoints = append(endpoints[:index], endpoints[index+1:]...)
	if len(endpoints) == 0 {
		c.nseByNs.Delete(nse.Spec.NetworkServiceName)
	} else {
		c.nseByNs.Store(nse.Spec.NetworkServiceName, endpoints)
	}
	c.networkServiceEndpoints.Delete(key)
}

func getNseKey(obj interface{}) string {
	return obj.(*v1.NetworkServiceEndpoint).Name
}
