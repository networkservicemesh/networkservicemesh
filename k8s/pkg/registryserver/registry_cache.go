package registryserver

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	nsmClientset "github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
	"time"
)

const (
	nseResource = "networkserviceendpoints"
	nsResource  = "networkservices"
	nsmResource = "networkservicemanagers"
)

type RegistryCache interface {
	GetNetworkService(name string) (*v1.NetworkService, error)
	GetNetworkServiceManager(name string) (*v1.NetworkServiceManager, error)
	GetNetworkServiceEndpoints(networkServiceName string) ([]*v1.NetworkServiceEndpoint, error)
	Close() error
}

type registryCacheImpl struct {
	stores map[string]cache.Store
}

func NewRegistryCache(clientSet *nsmClientset.Clientset) (RegistryCache, error) {
	factory := externalversions.NewSharedInformerFactory(clientSet, 100*time.Millisecond)
	resources := []string{nseResource, nsResource, nsmResource}
	stores := map[string]cache.Store{}

	for _, resource := range resources {
		genericInformer, err := factory.ForResource(v1.SchemeGroupVersion.WithResource(resource))
		if err != nil {
			return nil, err
		}
		informer := genericInformer.Informer()
		if resource == nsmResource {
			informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					nsm := obj.(*v1.NetworkServiceManager)
					logrus.Infof("NSM added: %v", nsm.Name)
				},
			})
		}

		stopper := make(chan struct{})

		go informer.Run(stopper)
		stores[resource] = informer.GetStore()
	}

	return &registryCacheImpl{
		stores: stores,
	}, nil
}

func (rc *registryCacheImpl) GetNetworkService(name string) (*v1.NetworkService, error) {
	item, exist, err := rc.stores[nsResource].GetByKey(name)
	service, cast := item.(*v1.NetworkService)
	if err != nil {
		return nil, err
	}
	if !exist || !cast {
		return nil, fmt.Errorf("failed to get NetworkService with name: %v", name)
	}
	return service, nil
}
func (rc *registryCacheImpl) GetNetworkServiceEndpoints(networkServiceName string) ([]*v1.NetworkServiceEndpoint, error) {
	//todo (lobkovilya): use map to avoid linear search
	var rv []*v1.NetworkServiceEndpoint
	for _, item := range rc.stores[nseResource].List() {
		nse := item.(*v1.NetworkServiceEndpoint)
		if nse.Spec.NetworkServiceName == networkServiceName {
			rv = append(rv, nse)
		}
	}
	return rv, nil
}

func (rc *registryCacheImpl) GetNetworkServiceManager(name string) (*v1.NetworkServiceManager, error) {
	item, exist, err := rc.stores[nsmResource].GetByKey(name)

	logrus.Infof("rc.stores[nsmResource].List(): %v", rc.stores[nsmResource].List())
	logrus.Infof("rc.stores[nsmResource].ListKeys(): %v", rc.stores[nsmResource].ListKeys())

	nsm, cast := item.(*v1.NetworkServiceManager)
	if err != nil {
		return nil, err
	}
	if !exist || !cast {
		return nil, fmt.Errorf("failed to get NetworkServiceManager with name: %v", name)
	}
	return nsm, nil
}

func (rc *registryCacheImpl) Close() error {
	//todo (lobkovilya): implement close
	return nil
}
