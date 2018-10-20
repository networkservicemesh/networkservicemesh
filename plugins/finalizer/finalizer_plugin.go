// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package finalizer

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ligato/cn-infra/health/statuscheck"
	nsmclient "github.com/ligato/networkservicemesh/pkg/client/clientset/versioned"
	"github.com/ligato/networkservicemesh/plugins/k8sclient"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	"github.com/ligato/networkservicemesh/utils/helper/plugintools"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	"github.com/ligato/networkservicemesh/utils/registry"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	// Label to select pods treated by NSM controller
	nsmLabel     = "networkservicemesh.io"
	nsmAppLabel  = "networkservicemesh.io/app"
	nsmAppNSE    = "nse"
	nsmAppClient = "nsm-client"
	// NSMFinalizer defines a finalizer which is add to every nsm client after successful programming
	// of its dataplane.
	NSMFinalizer = "nsm-client.networkservicemesh.io/nsm"
	// NSEFinalizerSuffix is a part of the finalizer name nsm cient pod name + this suffix
	// it allows to track all clients using the specific nse
	NSEFinalizerSuffix = ".networkservicemesh.io/nsm"
	// EndpointFinalizer is a finalizer added to each nse pod whose endpoint
	// was successfully advertised. It allows to clean up endpoints advertised for the nse pod
	// before it will be deleted.
	EndpointFinalizer = "endpoints.networkservicemesh.io/nsm"
)

// Plugin watches K8s resources and causes all changes to be reflected in the ETCD
// data store.
type Plugin struct {
	idempotent.Impl
	Deps
	pluginStopCh  chan struct{}
	wg            sync.WaitGroup
	StatusMonitor statuscheck.StatusReader
	stopCh        chan struct{}
	informer      cache.SharedIndexInformer
	k8sClient     kubernetes.Interface
	nsmClient     nsmclient.Interface
	namespace     string
}

// Deps defines dependencies of CRD plugin.
type Deps struct {
	Name        string
	Log         logger.FieldLoggerPlugin
	ObjectStore objectstore.Interface
	K8sclient   k8sclient.API
}

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (plugin *Plugin) Init() error {
	return plugin.IdempotentInit(plugintools.LoggingInitFunc(plugin.Log, plugin, plugin.init))
}

func (plugin *Plugin) init() error {
	plugin.pluginStopCh = make(chan struct{})
	err := deptools.Init(plugin)
	if err != nil {
		return err
	}
	plugin.k8sClient = plugin.K8sclient.GetClientset()
	plugin.nsmClient = plugin.K8sclient.GetNSMClientset()
	plugin.stopCh = make(chan struct{})
	// Getting NSM's Namespace
	plugin.namespace = os.Getenv("NSM_NAMESPACE")
	if plugin.namespace == "" {
		return fmt.Errorf("cannot detect namespace, make sure NSM_NAMESPACE variable is set via downward api")
	}
	return plugin.afterInit()
}

func setupInformer(plugin *Plugin) {
	plugin.informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldPod := oldObj.(*v1.Pod)
				newPod := newObj.(*v1.Pod)
				// This condition should be triggered only once when pod delete operation is initiated.
				if oldPod.ObjectMeta.DeletionTimestamp == nil && newPod.ObjectMeta.DeletionTimestamp != nil {
					plugin.CleanUp(newPod)
				}
			},
		},
	)
}

func (plugin *Plugin) afterInit() error {
	var err error

	err = nil
	if err != nil {
		plugin.Log.Error("Error initializing Finalizer plugin")
		return err
	}

	plugin.informer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				selector := labels.SelectorFromSet(labels.Set(map[string]string{nsmLabel: "true"}))
				options = metav1.ListOptions{LabelSelector: selector.String()}
				return plugin.k8sClient.CoreV1().Pods(plugin.namespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				selector := labels.SelectorFromSet(labels.Set(map[string]string{nsmLabel: "true"}))
				options = metav1.ListOptions{LabelSelector: selector.String()}
				return plugin.k8sClient.CoreV1().Pods(plugin.namespace).Watch(options)
			},
		},
		&v1.Pod{},
		60*time.Second,
		cache.Indexers{},
	)

	setupInformer(plugin)
	go plugin.informer.Run(plugin.stopCh)
	plugin.Log.Info("Started  Finalizer's shared informer factory.")

	// Wait for the informer caches to finish performing it's initial sync of
	// resources
	if !cache.WaitForCacheSync(plugin.stopCh, plugin.informer.HasSynced) {
		plugin.Log.Error("Error waiting for informer cache to sync")
	}
	plugin.Log.Info("Finalizer's Informer cache is ready")

	return nil
}

// Close stops all reflectors.
func (plugin *Plugin) Close() error {
	return plugin.IdempotentClose(plugintools.LoggingCloseFunc(plugin.Log, plugin, plugin.close))
}

func (plugin *Plugin) close() error {
	plugin.Log.Info("Close")
	close(plugin.pluginStopCh)
	plugin.wg.Wait()
	registry.Shared().Delete(plugin)
	return deptools.Close(plugin)
}
