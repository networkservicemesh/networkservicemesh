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

type NetworkServiceCache struct {
	cache           abstractResourceCache
	networkServices map[string]*v1.NetworkService
	getCh           chan *v1.NetworkService
}

//NewNetworkServiceCache creates cache for network services
func NewNetworkServiceCache(policy CacheFilterPolicy) *NetworkServiceCache {
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
	rv.cache = newAbstractResourceCache(config, policy)
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

func (c *NetworkServiceCache) Start(f SharedInformerFactory, init ...v1.NetworkService) (func(), error) {
	c.replace(init)
	return c.cache.start(f)
}

func (c *NetworkServiceCache) StartWithResync(f SharedInformerFactory, cs *versioned.Clientset) (func(), error) {
	l, err := cs.NetworkservicemeshV1alpha1().NetworkServices(namespace.GetNamespace()).List(v12.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list NSs for cache initialization: %v", err)
	}
	return c.Start(f, l.Items...)
}

func (c *NetworkServiceCache) replace(resources []v1.NetworkService) {
	c.networkServices = map[string]*v1.NetworkService{}
	logrus.Infof("Replacing Network services with: %v", resources)
	for i := 0; i < len(resources); i++ {
		c.resourceAdded(&resources[i])
	}
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
