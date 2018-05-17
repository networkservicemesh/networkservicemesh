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

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	client "github.com/ligato/networkservicemesh/pkg/client/clientset/versioned"
	factory "github.com/ligato/networkservicemesh/pkg/client/informers/externalversions"
)

// Plugin watches K8s resources and causes all changes to be reflected in the ETCD
// data store.
type Plugin struct {
	Deps

	stopCh chan struct{}
	wg     sync.WaitGroup

	k8sClientConfig *rest.Config
	k8sClientset    *kubernetes.Clientset
	apiclientset    *apiextcs.Clientset
	crdClient       client.Interface

	StatusMonitor statuscheck.StatusReader
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	local.PluginInfraDeps
	// Kubeconfig with k8s cluster address and access credentials to use.
	KubeConfig config.PluginConfig
}

var (
	// queue is a queue of resources to be processed. It performs exponential
	// backoff rate limiting, with a minimum retry period of 5 seconds and a
	// maximum of 1 minute.
	queueNS  = workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(time.Second*5, time.Minute))
	queueNSC = workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(time.Second*5, time.Minute))
	queueNSE = workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(time.Second*5, time.Minute))

	// stopCh can be used to stop all the informer, as well as control loops
	// within the application.
	stopCh = make(chan struct{})

	// sharedFactory is a shared informer factory that is used a a cache for
	// items in the API server. It saves each informer listing and watching the
	// same resources independently of each other, thus providing more up to
	// date results with less 'effort'
	sharedFactoryNS  factory.SharedInformerFactory
	sharedFactoryNSE factory.SharedInformerFactory
	sharedFactoryNSC factory.SharedInformerFactory
)

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (plugin *Plugin) Init() error {
	var err error
	plugin.Log.SetLevel(logging.DebugLevel)
	plugin.stopCh = make(chan struct{})

	kubeconfig := plugin.KubeConfig.GetConfigName()
	plugin.Log.WithField("kubeconfig", kubeconfig).Info("Loading kubernetes client config")
	plugin.k8sClientConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client config: %s", err)
	}

	plugin.k8sClientset, err = kubernetes.NewForConfig(plugin.k8sClientConfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %s", err)
	}

	return nil
}

// Create the CRD resource, ignore error if it already exists
func createCRD(plugin *Plugin, FullCRDName, CRDGroup, CRDVersion, CRDPlural, CRDName string) error {
	crd := &apiextv1beta1.CustomResourceDefinition{
		ObjectMeta: meta_v1.ObjectMeta{Name: FullCRDName},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:   CRDGroup,
			Version: CRDVersion,
			Scope:   apiextv1beta1.NamespaceScoped,
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Plural: CRDPlural,
				Kind:   CRDName,
			},
		},
	}

	_, cserr := plugin.apiclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if cserr != nil && apierrors.IsAlreadyExists(cserr) {
		plugin.Log.Infof("Created CRD %s succesfully, though it already existed", CRDName)
		return nil
	} else if cserr != nil {
		plugin.Log.Infof("Error creating CRD %s: %s", CRDName, cserr)
	} else {
		plugin.Log.Infof("Created CRD %s succesfully", CRDName)
	}

	return cserr
}

// Delete the CRD resource
// Note this is not currently used, as once we create the CRDs we want to leave them
// in the DB even after the plugin is closed.
func deleteCRD(plugin *Plugin, CRDName string) error {
	err := plugin.apiclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(CRDName, nil)
	if err != nil {
		plugin.Log.Infof("Error deleting CRD %s: %s", CRDName, err)
	} else {
		plugin.Log.Infof("Successfully deleted CRD %s", CRDName)
	}

	return err
}

func informerNetworkservices(plugin *Plugin) {
	var err error

	if err != nil {
		plugin.Log.Errorf("Error creating api client: %s", err.Error())
	}

	// We use a shared informer from the informer factory, to save calls to the
	// API as we grow our application and so state is consistent between our
	// control loops. We set a resync period of 30 seconds, in case any
	// create/replace/update/delete operations are missed when watching
	sharedFactoryNS = factory.NewSharedInformerFactory(plugin.crdClient, time.Second*30)

	informer := sharedFactoryNS.Networkservice().V1().NetworkServices().Informer()
	// We add a new event handler, watching for changes to API resources.
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: networkserviceEnqueue,
			UpdateFunc: func(old, cur interface{}) {
				if !reflect.DeepEqual(old, cur) {
					networkserviceEnqueue(cur)
				}
			},
			DeleteFunc: networkserviceEnqueue,
		},
	)

	// Start the informer. This will cause it to begin receiving updates from
	// the configured API server and firing event handlers in response.
	sharedFactoryNS.Start(stopCh)
	plugin.Log.Info("Started NetworkService informer factory.")

	// Wait for the informer cache to finish performing it's initial sync of
	// resources
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		plugin.Log.Errorf("Error waiting for informer cache to sync: %s", err.Error())
	}

	plugin.Log.Info("NetworkService Informer is ready")
}

func informerNetworkservicechannels(plugin *Plugin) {
	var err error

	if err != nil {
		plugin.Log.Errorf("Error creating api client: %s", err.Error())
	}

	// We use a shared informer from the informer factory, to save calls to the
	// API as we grow our application and so state is consistent between our
	// control loops. We set a resync period of 30 seconds, in case any
	// create/replace/update/delete operations are missed when watching
	sharedFactoryNSC = factory.NewSharedInformerFactory(plugin.crdClient, time.Second*30)

	informer := sharedFactoryNSC.Networkservice().V1().NetworkServiceChannels().Informer()
	// we add a new event handler, watching for changes to API resources.
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: networkservicechannelEnqueue,
			UpdateFunc: func(old, cur interface{}) {
				if !reflect.DeepEqual(old, cur) {
					networkservicechannelEnqueue(cur)
				}
			},
			DeleteFunc: networkservicechannelEnqueue,
		},
	)

	// Start the informer. This will cause it to begin receiving updates from
	// the configured API server and firing event handlers in response.
	sharedFactoryNSC.Start(stopCh)
	plugin.Log.Info("Started NetworkServiceChannel informer factory.")

	// Wait for the informer cache to finish performing it's initial sync of
	// resources
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		plugin.Log.Errorf("Error waiting for informer cache to sync: %s", err.Error())
	}

	plugin.Log.Info("NetworkServiceChannel Informer is ready")
}

func informerNetworkserviceendpoints(plugin *Plugin) {
	var err error

	if err != nil {
		plugin.Log.Errorf("Error creating api client: %s", err.Error())
	}

	// We use a shared informer from the informer factory, to save calls to the
	// API as we grow our application and so state is consistent between our
	// control loops. We set a resync period of 30 seconds, in case any
	// create/replace/update/delete operations are missed when watching
	sharedFactoryNSE = factory.NewSharedInformerFactory(plugin.crdClient, time.Second*30)

	informer := sharedFactoryNSE.Networkservice().V1().NetworkServiceEndpoints().Informer()
	// we add a new event handler, watching for changes to API resources.
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: networkserviceendpointEnqueue,
			UpdateFunc: func(old, cur interface{}) {
				if !reflect.DeepEqual(old, cur) {
					networkserviceendpointEnqueue(cur)
				}
			},
			DeleteFunc: networkserviceendpointEnqueue,
		},
	)

	// Start the informer. This will cause it to begin receiving updates from
	// the configured API server and firing event handlers in response.
	sharedFactoryNSE.Start(stopCh)
	plugin.Log.Info("Started NetworkServiceEndpoints informer factory.")

	// Wait for the informer cache to finish performing it's initial sync of
	// resources
	if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
		plugin.Log.Errorf("Error waiting for informer cache to sync: %s", err.Error())
	}

	plugin.Log.Info("NetworkServiceEndpoint Informer is ready")
}

// This will create all of the CRDs for NetworkServiceMesh.
func (plugin *Plugin) AfterInit() error {
	var err error
	var crdname string

	// create clientset and create our CRD, this only needs to run once
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

	go informerNetworkservices(plugin)
	go informerNetworkservicechannels(plugin)
	go informerNetworkserviceendpoints(plugin)
	go networkserviceWork(plugin)
	go networkservicechannelWork(plugin)
	go networkserviceendpointWork(plugin)

	return nil
}

// Close stops all reflectors.
func (plugin *Plugin) Close() error {
	close(plugin.stopCh)
	plugin.wg.Wait()
	return nil
}
