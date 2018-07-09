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

package netmeshplugincrd

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	client "github.com/ligato/networkservicemesh/pkg/client/clientset/versioned"
	factory "github.com/ligato/networkservicemesh/pkg/client/informers/externalversions"
	"github.com/ligato/networkservicemesh/plugins/handler"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"github.com/ligato/networkservicemesh/utils/command"
	"github.com/ligato/networkservicemesh/utils/idempotent"
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
	apiclientset    *apiextcs.Clientset
	crdClient       client.Interface
	StatusMonitor   statuscheck.StatusReader

	// These can be used to stop all the informers, as well as control loops
	// within the application.
	stopChNS  chan struct{}
	stopChNSE chan struct{}
	stopChNSC chan struct{}
	// sharedFactory is a shared informer factory used as a cache for
	// items in the API server. It saves each informer listing and watches the
	// same resources independently of each other, thus providing more up to
	// date results with less 'effort'
	sharedFactory factory.SharedInformerFactory

	// Informer factories per CRD object
	informerNS  cache.SharedIndexInformer
	informerNSE cache.SharedIndexInformer
	informerNSC cache.SharedIndexInformer
}

// Deps defines dependencies of CRD plugin.
type Deps struct {
	Name string
	Log  logger.FieldLoggerPlugin
	// Kubeconfig with k8s cluster address and access credentials to use.
	KubeConfig  string
	Handler     handler.PluginAPI
	ObjectStore objectstore.PluginAPI
}

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (plugin *Plugin) Init() error {
	return plugin.IdempotentInit(plugin.init)
}
func (plugin *Plugin) init() error {
	plugin.pluginStopCh = make(chan struct{})
	err := plugin.Log.Init()
	if err != nil {
		return err
	}
	err = plugin.Handler.Init()
	if err != nil {
		return err
	}
	err = plugin.ObjectStore.Init()
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

	plugin.stopChNS = make(chan struct{})
	plugin.stopChNSC = make(chan struct{})
	plugin.stopChNSE = make(chan struct{})

	return plugin.afterInit()
}

func setupInformer(informer cache.SharedIndexInformer, queue workqueue.RateLimitingInterface) {
	// We add a new event handler, watching for changes to API resources.
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				var message objectMessage
				var err error
				message.key, err = cache.MetaNamespaceKeyFunc(obj)
				message.operation = createOp
				message.obj = obj
				if err == nil {
					queue.Add(message)
				}
			},
			UpdateFunc: func(old, cur interface{}) {
				if !reflect.DeepEqual(old, cur) {
					// For an update event, we delete the old and add the current
					var messageOld, messageCur objectMessage
					var err error
					messageOld.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(old)
					messageOld.operation = deleteOp
					messageOld.obj = old
					if err == nil {
						queue.Add(messageOld)
					}
					messageCur.key, err = cache.MetaNamespaceKeyFunc(cur)
					messageCur.operation = createOp
					messageCur.obj = cur
					if err == nil {
						queue.Add(messageCur)
					}
				}
			},
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

// afterInit This will create all of the CRDs for NetworkServiceMesh.
func (plugin *Plugin) afterInit() error {
	var err error
	var crdname string

	// Create clientset and create our CRD, this only needs to run once
	plugin.apiclientset, err = apiextcs.NewForConfig(plugin.k8sClientConfig)
	if err != nil {
		panic(err.Error())
	}

	// Create an instance of our own API client
	plugin.crdClient, err = client.NewForConfig(plugin.k8sClientConfig)
	if err != nil {
		plugin.Log.Errorf("Error creating CRD client: %s", err.Error())
		panic(err.Error())
	}

	crdname = reflect.TypeOf(v1.NetworkServiceEndpoint{}).Name()
	err = createCRD(plugin, v1.FullNSMEPName,
		v1.NSMGroup,
		v1.NSMGroupVersion,
		v1.NSMEPPlural,
		crdname)

	if err != nil {
		plugin.Log.Error("Error initializing NetworkServiceEndpoint CRD")
		return err
	}

	crdname = reflect.TypeOf(v1.NetworkServiceChannel{}).Name()
	err = createCRD(plugin, v1.FullNSMChannelName,
		v1.NSMGroup,
		v1.NSMGroupVersion,
		v1.NSMChannelPlural,
		crdname)

	if err != nil {
		plugin.Log.Error("Error initializing NetworkServiceChannel CRD")
		return err
	}

	crdname = reflect.TypeOf(v1.NetworkService{}).Name()
	err = createCRD(plugin, v1.FullNSMName,
		v1.NSMGroup,
		v1.NSMGroupVersion,
		v1.NSMPlural,
		crdname)

	if err != nil {
		plugin.Log.Error("Error initializing NetworkService CRD")
		return err
	}

	// We use a shared informer from the informer factory, to save calls to the
	// API as we grow our application and so state is consistent between our
	// control loops. We set a resync period of 30 seconds, in case any
	// create/replace/update/delete operations are missed when watching
	plugin.sharedFactory = factory.NewSharedInformerFactory(plugin.crdClient, time.Second*30)

	plugin.informerNS = plugin.sharedFactory.Networkservice().V1().NetworkServices().Informer()
	setupInformer(plugin.informerNS, queueNS)
	plugin.informerNSC = plugin.sharedFactory.Networkservice().V1().NetworkServiceChannels().Informer()
	setupInformer(plugin.informerNSC, queueNSC)
	plugin.informerNSE = plugin.sharedFactory.Networkservice().V1().NetworkServiceEndpoints().Informer()
	setupInformer(plugin.informerNSE, queueNSE)

	// Start the informer. This will cause it to begin receiving updates from
	// the configured API server and firing event handlers in response.
	plugin.sharedFactory.Start(plugin.stopChNS)
	plugin.Log.Info("Started shared informer factory.")

	// Wait for the informer caches to finish performing it's initial sync of
	// resources
	if !cache.WaitForCacheSync(plugin.stopChNS, plugin.informerNS.HasSynced, plugin.informerNSC.HasSynced, plugin.informerNSE.HasSynced) {
		plugin.Log.Error("Error waiting for informer cache to sync")
	}
	plugin.Log.Info("Informer cache is ready")

	// Read forever from the work queue
	go workforever(plugin, queueNS, plugin.informerNS, plugin.stopChNS)
	go workforever(plugin, queueNSC, plugin.informerNSC, plugin.stopChNSC)
	go workforever(plugin, queueNSE, plugin.informerNSE, plugin.stopChNSE)

	return nil
}

// Close stops all reflectors.
func (plugin *Plugin) Close() error {
	return plugin.IdempotentClose(plugin.close)
}

func (plugin *Plugin) close() error {
	close(plugin.pluginStopCh)
	plugin.wg.Wait()
	err := plugin.Deps.ObjectStore.Close()
	if err != nil {
		return err
	}
	err = plugin.Deps.Handler.Close()
	if err != nil {
		return err
	}
	err = plugin.Deps.Log.Close()
	if err != nil {
		return err
	}
	return nil
}
