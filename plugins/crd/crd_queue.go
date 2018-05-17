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

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
)

// This file contains the work queue backends for each of the CRDs we create in the
// plugin AfterInit() call.

func networkserviceWork(plugin *Plugin) {
	for {
		// we read a message off the queue
		key, shutdown := queueNS.Get()

		// if the queue has been shut down, we should exit the work queue here
		if shutdown {
			stopCh <- struct{}{}
			return
		}

		// convert the queue item into a string. If it's not a string, we'll
		// simply discard it as invalid data and log a message.
		var strKey string
		var ok bool
		if strKey, ok = key.(string); !ok {
			runtime.HandleError(fmt.Errorf("key in queue should be of type string but got %T. discarding", key))
			return
		}

		// we define a function here to process a queue item, so that we can
		// use 'defer' to make sure the message is marked as Done on the queue
		func(key string) {
			var obj interface{}
			var err error

			defer queueNS.Done(key)

			// attempt to split the 'key' into namespace and object name
			namespace, name, err := cache.SplitMetaNamespaceKey(strKey)

			if err != nil {
				runtime.HandleError(fmt.Errorf("error splitting meta namespace key into parts: %s", err.Error()))
				return
			}

			plugin.Log.Infof("Read item '%s/%s' off workqueue. Processing...", namespace, name)

			// retrieve the latest version in the cache of this alert
			plugin.Log.Infof("Dequeuing interface of type %s", v1.NSMPlural)
			obj, err = sharedFactoryNS.Networkservice().V1().NetworkServices().Lister().NetworkServices(namespace).Get(name)

			if err != nil {
				runtime.HandleError(fmt.Errorf("error getting object '%s/%s' from api: %s", namespace, name, err.Error()))
				return
			}

			plugin.Log.Infof("Got most up to date version of '%s/%s'. Syncing...", namespace, name)
			plugin.Log.Infof("Object found: %s", obj)
			plugin.Log.Infof("Finished processing '%s/%s' successfully! Removing from queue.", namespace, name)

			// as we managed to process this successfully, we can forget it
			// from the work queue altogether.
			queueNS.Forget(key)
		}(strKey)
	}
}

// networkservice_enqueue will add an object 'obj' into the workqueue. The object being added
// must be of type metav1.Object, metav1.ObjectAccessor or cache.ExplicitKey.
func networkserviceEnqueue(obj interface{}) {
	// DeletionHandlingMetaNamespaceKeyFunc will convert an object into a
	// 'namespace/name' string. We do this because our item may be processed
	// much later than now, and so we want to ensure it gets a fresh copy of
	// the resource when it starts. Also, this allows us to keep adding the
	// same item into the work queue without duplicates building up.
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("error obtaining key for object being enqueue: %s", err.Error()))
		return
	}
	// add the item to the queue
	queueNS.Add(key)
}

func networkservicechannelWork(plugin *Plugin) {
	for {
		// we read a message off the queue
		key, shutdown := queueNSC.Get()

		// if the queue has been shut down, we should exit the work queue here
		if shutdown {
			stopCh <- struct{}{}
			return
		}

		// convert the queue item into a string. If it's not a string, we'll
		// simply discard it as invalid data and log a message.
		var strKey string
		var ok bool
		if strKey, ok = key.(string); !ok {
			runtime.HandleError(fmt.Errorf("key in queue should be of type string but got %T. discarding", key))
			return
		}

		// we define a function here to process a queue item, so that we can
		// use 'defer' to make sure the message is marked as Done on the queue
		func(key string) {
			var obj interface{}
			var err error

			defer queueNSC.Done(key)

			// attempt to split the 'key' into namespace and object name
			namespace, name, err := cache.SplitMetaNamespaceKey(strKey)

			if err != nil {
				runtime.HandleError(fmt.Errorf("error splitting meta namespace key into parts: %s", err.Error()))
				return
			}

			plugin.Log.Infof("Read item '%s/%s' off workqueue. Processing...", namespace, name)

			// retrieve the latest version in the cache of this alert
			plugin.Log.Infof("Dequeuing interface of type %s", v1.NSMChannelPlural)
			obj, err = sharedFactoryNS.Networkservice().V1().NetworkServiceChannels().Lister().NetworkServiceChannels(namespace).Get(name)

			if err != nil {
				runtime.HandleError(fmt.Errorf("error getting object '%s/%s' from api: %s", namespace, name, err.Error()))
				return
			}

			plugin.Log.Infof("Got most up to date version of '%s/%s'. Syncing...", namespace, name)
			plugin.Log.Infof("Object found: %s", obj)
			plugin.Log.Infof("Finished processing '%s/%s' successfully! Removing from queue.", namespace, name)

			// as we managed to process this successfully, we can forget it
			// from the work queue altogether.
			queueNSC.Forget(key)
		}(strKey)
	}
}

// networkservicechannel_enqueue will add an object 'obj' into the workqueue. The object being added
// must be of type metav1.Object, metav1.ObjectAccessor or cache.ExplicitKey.
func networkservicechannelEnqueue(obj interface{}) {
	// DeletionHandlingMetaNamespaceKeyFunc will convert an object into a
	// 'namespace/name' string. We do this because our item may be processed
	// much later than now, and so we want to ensure it gets a fresh copy of
	// the resource when it starts. Also, this allows us to keep adding the
	// same item into the work queue without duplicates building up.
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("error obtaining key for object being enqueue: %s", err.Error()))
		return
	}
	// add the item to the queue
	queueNSC.Add(key)
}

func networkserviceendpointWork(plugin *Plugin) {
	for {
		// we read a message off the queue
		key, shutdown := queueNSE.Get()

		// if the queue has been shut down, we should exit the work queue here
		if shutdown {
			stopCh <- struct{}{}
			return
		}

		// convert the queue item into a string. If it's not a string, we'll
		// simply discard it as invalid data and log a message.
		var strKey string
		var ok bool
		if strKey, ok = key.(string); !ok {
			runtime.HandleError(fmt.Errorf("key in queue should be of type string but got %T. discarding", key))
			return
		}

		// we define a function here to process a queue item, so that we can
		// use 'defer' to make sure the message is marked as Done on the queue
		func(key string) {
			var obj interface{}
			var err error

			defer queueNSE.Done(key)

			// attempt to split the 'key' into namespace and object name
			namespace, name, err := cache.SplitMetaNamespaceKey(strKey)

			if err != nil {
				runtime.HandleError(fmt.Errorf("error splitting meta namespace key into parts: %s", err.Error()))
				return
			}

			plugin.Log.Infof("Read item '%s/%s' off workqueue. Processing...", namespace, name)

			// retrieve the latest version in the cache of this alert
			plugin.Log.Infof("Dequeuing interface of type %s", v1.NSMEPPlural)
			obj, err = sharedFactoryNS.Networkservice().V1().NetworkServiceEndpoints().Lister().NetworkServiceEndpoints(namespace).Get(name)

			if err != nil {
				runtime.HandleError(fmt.Errorf("error getting object '%s/%s' from api: %s", namespace, name, err.Error()))
				return
			}

			plugin.Log.Infof("Got most up to date version of '%s/%s'. Syncing...", namespace, name)
			plugin.Log.Infof("Object found: %s", obj)
			plugin.Log.Infof("Finished processing '%s/%s' successfully! Removing from queue.", namespace, name)

			// as we managed to process this successfully, we can forget it
			// from the work queue altogether.
			queueNSE.Forget(key)
		}(strKey)
	}
}

// networkserviceendpoint_enqueue will add an object 'obj' into the workqueue. The object being added
// must be of type metav1.Object, metav1.ObjectAccessor or cache.ExplicitKey.
func networkserviceendpointEnqueue(obj interface{}) {
	// DeletionHandlingMetaNamespaceKeyFunc will convert an object into a
	// 'namespace/name' string. We do this because our item may be processed
	// much later than now, and so we want to ensure it gets a fresh copy of
	// the resource when it starts. Also, this allows us to keep adding the
	// same item into the work queue without duplicates building up.
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("error obtaining key for object being enqueue: %s", err.Error()))
		return
	}
	// add the item to the queue
	queueNSE.Add(key)
}
