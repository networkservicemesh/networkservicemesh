package tests

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"

	"github.com/sirupsen/logrus"

	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
)

type fakeRegistry struct {
	externalversions.SharedInformerFactory
	externalversions.GenericInformer
	cache.SharedIndexInformer

	eventHandlers []cache.ResourceEventHandler
}

func (f *fakeRegistry) Run(stopCh <-chan struct{}) {

}

func (f *fakeRegistry) ForResource(resource schema.GroupVersionResource) (externalversions.GenericInformer, error) {
	return f, nil
}

func (f *fakeRegistry) Informer() cache.SharedIndexInformer {
	return f
}

func (f *fakeRegistry) AddEventHandler(handler cache.ResourceEventHandler) {
	f.eventHandlers = append(f.eventHandlers, handler)
}

func (f *fakeRegistry) Add(obj interface{}) {
	logrus.Info(len(f.eventHandlers))
	for _, eh := range f.eventHandlers {
		eh.OnAdd(obj)
	}
}

func (f *fakeRegistry) Delete(nse *v1.NetworkServiceEndpoint) {
	for _, eh := range f.eventHandlers {
		eh.OnDelete(nse)
	}
}
