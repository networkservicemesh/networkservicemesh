package resource_cache

import (
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/sirupsen/logrus"
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
	resourceUpdatedFunc func(obj interface{})
	resourceDeletedFunc func(key string)
	resourceType        string
}

type abstractResourceCache struct {
	addCh    chan interface{}
	deleteCh chan string
	updateCh chan interface{}
	config   cacheConfig
}

const defaultChannelSize = 10

func newAbstractResourceCache(config cacheConfig) abstractResourceCache {
	return abstractResourceCache{
		addCh:    make(chan interface{}, defaultChannelSize),
		deleteCh: make(chan string, defaultChannelSize),
		updateCh: make(chan interface{}, defaultChannelSize),
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

func (c *abstractResourceCache) update(obj interface{}) {
	c.updateCh <- obj
}

func (c *abstractResourceCache) delete(key string) {
	c.deleteCh <- key
}

func (c *abstractResourceCache) run(stopCh chan struct{}) {
	for {
		select {
		case newResource := <-c.addCh:
			if c.config.resourceAddedFunc != nil {
				c.config.resourceAddedFunc(newResource)
			}
		case upd := <-c.updateCh:
			if c.config.resourceUpdatedFunc != nil {
				c.config.resourceUpdatedFunc(upd)
			}
		case deleteResourceKey := <-c.deleteCh:
			if c.config.resourceDeletedFunc != nil {
				c.config.resourceDeletedFunc(deleteResourceKey)
			}
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
		AddFunc: c.add,
		UpdateFunc: func(old interface{}, new interface{}) {
			if _, ok := old.(*v1.NetworkServiceManager); !ok {
				return
			}
			logrus.Info("Update from k8s-registry: ")
			logrus.Infof("Old NSM: %v", old.(*v1.NetworkServiceManager))
			logrus.Infof("New NSM: %v", new.(*v1.NetworkServiceManager))
			c.update(new)
		},
		DeleteFunc: func(obj interface{}) { c.delete(c.config.keyFunc(obj)) },
	})

	stopCh := make(chan struct{})
	go informer.Run(stopCh)
	return stopCh, nil
}
