package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

const (
	NseResource = "networkserviceendpoints"
	NsResource  = "networkservices"
	NsmResource = "networkservicemanagers"
)

type cacheConfig struct {
	keyFunc             func(obj interface{}) string
	resourceAddedFunc   func(obj interface{})
	resourceDeletedFunc func(key string)
	resourceType        string
}

type abstractResourceCache struct {
	addCh    chan interface{}
	deleteCh chan string
	config   cacheConfig
}

func newAbstractResourceCache(config cacheConfig) abstractResourceCache {
	return abstractResourceCache{
		addCh:    make(chan interface{}, 10),
		deleteCh: make(chan string, 10),
		config:   config,
	}
}

func (c *abstractResourceCache) start(informerFactory externalversions.SharedInformerFactory) (func(), error) {
	informerStopCh, err := c.startInformer(informerFactory)
	if err != nil {
		return nil, err
	}
	cacheStopCh := make(chan struct{})
	go c.run(cacheStopCh)
	stopFunc := func() {
		close(informerStopCh)
		close(cacheStopCh)
	}
	return stopFunc, nil
}

func (c *abstractResourceCache) add(obj interface{}) {
	c.addCh <- obj
}

func (c *abstractResourceCache) delete(key string) {
	c.deleteCh <- key
}

func (c *abstractResourceCache) run(stopCh chan struct{}) {
	for {
		select {
		case newResource := <-c.addCh:
			c.config.resourceAddedFunc(newResource)
		case deleteResourceKey := <-c.deleteCh:
			c.config.resourceDeletedFunc(deleteResourceKey)
		case <-stopCh:
			return
		}
	}
}

func (c *abstractResourceCache) startInformer(informerFactory externalversions.SharedInformerFactory) (chan struct{}, error) {
	genericInformer, err := informerFactory.ForResource(v1.SchemeGroupVersion.WithResource(c.config.resourceType))
	if err != nil {
		return nil, err
	}

	informer := genericInformer.Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.add,
		DeleteFunc: func(obj interface{}) { c.delete(c.config.keyFunc(obj)) },
	})

	stopCh := make(chan struct{})
	go informer.Run(stopCh)
	return stopCh, nil
}
