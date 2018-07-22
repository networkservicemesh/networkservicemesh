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

// //go:generate protoc -I ./model/pod --go_out=plugins=grpc:./model/pod ./model/pod/pod.proto

package finalizer

import (
	"fmt"
	"sync"
	"time"

	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"github.com/ligato/networkservicemesh/utils/command"
	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

// Plugin watches K8s resources and causes all changes to be reflected in the ETCD
// data store.
type Plugin struct {
	idempotent.Impl
	Deps

	pluginStopCh    chan struct{}
	wg              sync.WaitGroup
	k8sClientConfig *rest.Config
	k8sClientset    *kubernetes.Clientset

	StatusMonitor statuscheck.StatusReader

	stopCh   chan struct{}
	informer cache.SharedIndexInformer
}

// Deps defines dependencies of CRD plugin.
type Deps struct {
	Name string
	Log  logger.FieldLoggerPlugin
	// Kubeconfig with k8s cluster address and access credentials to use.
	KubeConfig  string
	ObjectStore objectstore.Interface
}

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (plugin *Plugin) Init() error {
	return plugin.IdempotentInit(plugin.init)
}
func (plugin *Plugin) init() error {
	plugin.pluginStopCh = make(chan struct{})
	err := deptools.Init(plugin)
	if err != nil {
		return err
	}
	plugin.KubeConfig = command.RootCmd().Flags().Lookup(KubeConfigFlagName).Value.String()

	plugin.Log.WithField("kubeconfig", plugin.KubeConfig).Info("Loading kubernetes client config")
	plugin.k8sClientConfig, err = clientcmd.BuildConfigFromFlags("", plugin.KubeConfig)
	if err != nil {
		return fmt.Errorf("Failed to build kubernetes client config: %s", err)
	}

	plugin.k8sClientset, err = kubernetes.NewForConfig(plugin.k8sClientConfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %s", err)
	}

	plugin.stopCh = make(chan struct{})

	return plugin.afterInit()
}

func setupInformer(informer cache.SharedIndexInformer, queue workqueue.RateLimitingInterface) {
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				var message objectMessage
				var err error
				message.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				message.operation = deleteOp
				message.obj = obj
				if err == nil {
					queue.Add(message)
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

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	plugin.informer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return plugin.k8sClientset.CoreV1().Pods(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return plugin.k8sClientset.CoreV1().Pods(metav1.NamespaceAll).Watch(options)
			},
		},
		&v1.Pod{},
		10*time.Second,
		cache.Indexers{},
	)

	setupInformer(plugin.informer, queue)

	go plugin.informer.Run(plugin.stopCh)
	plugin.Log.Info("Started  Finalizer's shared informer factory.")

	// Wait for the informer caches to finish performing it's initial sync of
	// resources
	if !cache.WaitForCacheSync(plugin.stopCh, plugin.informer.HasSynced) {
		plugin.Log.Error("Error waiting for informer cache to sync")
	}
	plugin.Log.Info("Finalizer's Informer cache is ready")

	// Read forever from the work queue
	go workforever(plugin, queue, plugin.informer, plugin.stopCh)

	return nil
}

// Close stops all reflectors.
func (plugin *Plugin) Close() error {
	return plugin.IdempotentClose(plugin.close)
}

func (plugin *Plugin) close() error {
	close(plugin.pluginStopCh)
	plugin.wg.Wait()
	return deptools.Close(plugin)
}
