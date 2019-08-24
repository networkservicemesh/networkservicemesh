package resourcecache

import (
	"fmt"

	"github.com/sirupsen/logrus"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	. "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"
)

type NetworkServiceEndpointCache struct {
	cache                   abstractResourceCache
	nseByNs                 map[string][]*v1.NetworkServiceEndpoint
	networkServiceEndpoints map[string]*v1.NetworkServiceEndpoint
}

//NewNetworkServiceEndpointCache creates cache for network service endpoints
func NewNetworkServiceEndpointCache(policy CacheFilterPolicy) *NetworkServiceEndpointCache {
	rv := &NetworkServiceEndpointCache{
		nseByNs:                 make(map[string][]*v1.NetworkServiceEndpoint),
		networkServiceEndpoints: make(map[string]*v1.NetworkServiceEndpoint),
	}
	config := cacheConfig{
		keyFunc:             getNseKey,
		resourceAddedFunc:   rv.resourceAdded,
		resourceDeletedFunc: rv.resourceDeleted,
		resourceGetFunc:     rv.resourceGet,
		resourceType:        NseResource,
	}
	rv.cache = newAbstractResourceCache(config, policy)
	return rv
}

func (c *NetworkServiceEndpointCache) Get(key string) *v1.NetworkServiceEndpoint {
	v := c.cache.get(key)
	if v != nil {
		return v.(*v1.NetworkServiceEndpoint)
	}
	return nil
}

func (c *NetworkServiceEndpointCache) GetByNetworkService(networkServiceName string) []*v1.NetworkServiceEndpoint {
	var result []*v1.NetworkServiceEndpoint
	c.cache.syncExec(func() {
		result = c.nseByNs[networkServiceName]
	})
	return result
}

func (c *NetworkServiceEndpointCache) GetByNetworkServiceManager(nsmName string) []*v1.NetworkServiceEndpoint {
	var rv []*v1.NetworkServiceEndpoint
	c.cache.syncExec(func() {
		for _, endpoint := range c.networkServiceEndpoints {
			if endpoint.Spec.NsmName == nsmName {
				rv = append(rv, endpoint)
			}
		}
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

func (c *NetworkServiceEndpointCache) Start(f SharedInformerFactory, init ...v1.NetworkServiceEndpoint) (func(), error) {
	c.replace(init)
	return c.cache.start(f)
}

func (c *NetworkServiceEndpointCache) StartWithResync(f SharedInformerFactory, cs *versioned.Clientset) (func(), error) {
	l, err := cs.NetworkservicemeshV1alpha1().NetworkServiceEndpoints(namespace.GetNamespace()).List(v12.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list NSEs for cache initialization: %v", err)
	}
	return c.Start(f, l.Items...)
}

func (c *NetworkServiceEndpointCache) replace(resources []v1.NetworkServiceEndpoint) {
	c.networkServiceEndpoints = map[string]*v1.NetworkServiceEndpoint{}
	c.nseByNs = map[string][]*v1.NetworkServiceEndpoint{}
	logrus.Infof("Replacing Network service endpoints with: %v", resources)

	for i := 0; i < len(resources); i++ {
		c.resourceAdded(&resources[i])
	}
}

func (c *NetworkServiceEndpointCache) resourceAdded(obj interface{}) {
	nse := obj.(*v1.NetworkServiceEndpoint)
	endpoints := c.nseByNs[nse.Spec.NetworkServiceName]
	if _, exist := c.networkServiceEndpoints[getNseKey(nse)]; !exist {
		c.nseByNs[nse.Spec.NetworkServiceName] = append(endpoints, nse)
	} else {
		for i, e := range endpoints {
			if getNseKey(nse) == getNseKey(e) {
				endpoints[i] = nse
				break
			}
		}
	}
	c.networkServiceEndpoints[getNseKey(nse)] = nse
}

func (c *NetworkServiceEndpointCache) resourceDeleted(key string) {
	nse, exist := c.networkServiceEndpoints[key]
	if !exist {
		return
	}

	endpoints := c.nseByNs[nse.Spec.NetworkServiceName]
	var index int
	for i, e := range endpoints {
		if getNseKey(nse) == getNseKey(e) {
			index = i
			break
		}
	}
	endpoints = append(endpoints[:index], endpoints[index+1:]...)
	if len(endpoints) == 0 {
		delete(c.nseByNs, nse.Spec.NetworkServiceName)
	} else {
		c.nseByNs[nse.Spec.NetworkServiceName] = endpoints
	}
	delete(c.networkServiceEndpoints, key)
}

func getNseKey(obj interface{}) string {
	return obj.(*v1.NetworkServiceEndpoint).Name
}

func (c *NetworkServiceEndpointCache) resourceGet(key string) interface{} {
	return c.networkServiceEndpoints[key]
}
