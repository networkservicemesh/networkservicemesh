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
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// QueueRetryCount is the max number of times to retry processing a failed item
// from the workqueue.
const QueueRetryCount = 5

var (
	// queue is a queue of resources to be processed. It performs exponential
	// backoff rate limiting, with a minimum retry period of 5 seconds and a
	// maximum of 1 minute.
	queueNS  = workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(time.Second*5, time.Minute))
	queueNSC = workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(time.Second*5, time.Minute))
	queueNSE = workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(time.Second*5, time.Minute))
)

// This file contains the work queue backends for each of the CRDs we create in the
// plugin AfterInit() call.

func workforever(plugin *Plugin, queue workqueue.RateLimitingInterface, stopCH chan struct{}) {
	for {
		// We read a message off the queue ...
		key, shutdown := queue.Get()

		// If the queue has been shut down, we should exit the work queue here
		if shutdown {
			plugin.Log.Error("shutdown signaled, closing stopChNS")
			close(stopCH)
			return
		}

		// Convert the queue item into a string. If it's not a string, we'll
		// simply discard it as invalid data and log a message.
		var strKey string
		var ok bool
		if strKey, ok = key.(string); !ok {
			runtime.HandleError(fmt.Errorf("key in queue should be of type string but got %T. discarding", key))
			return
		}

		// We define a function here to process a queue item, so that we can
		// use 'defer' to make sure the message is marked as Done on the queue
		func(key string) {
			defer queue.Done(key)

			// Attempt to split the 'key' into namespace and object name
			namespace, name, err := cache.SplitMetaNamespaceKey(strKey)

			if err != nil {
				// This is a soft-error
				plugin.Log.Errorf("Error splitting meta namespace key into parts: %s", err.Error())
				queue.Forget(key)
				return
			}

			plugin.Log.Infof("Read item '%s/%s' off workqueue. Processing...", namespace, name)

			// Retrieve the latest version in the cache of this NetworkService. By using
			// GetByKey() we are able to determine if the item exists, and thus if it was
			// added or deleted from the queue, and process appropriately
			item, exists, err := plugin.informerNS.GetIndexer().GetByKey(strKey)

			if err != nil {
				if queue.NumRequeues(key) < QueueRetryCount {
					plugin.Log.Errorf("Requeueing after error processing item with key %s, error %v", key, err)
					queue.AddRateLimited(key)
					return
				}

				plugin.Log.Errorf("Failed processing item with key %s, error %v, no more retries", key, err)
				queue.Forget(key)
				return
			}

			// Verify if this was a delete vs. an add/update
			if !exists {
				plugin.Log.Infof("Object (%s) deleted from queue", name)
			} else {
				plugin.Log.Infof("Got most up to date version of '%s/%s'. Syncing...", namespace, name)
				plugin.Log.Infof("Object found: %s", item)
				plugin.Log.Infof("Finished processing '%s/%s' successfully! Removing from queue.", namespace, name)
			}

			// As we managed to process this successfully, we can forget it
			// from the work queue altogether.
			queue.Forget(key)
		}(strKey)
	}
}

// networkserviceEnqeue will add an object 'obj' into the workqueue. The object being added
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
	// Add the item to the queue
	queueNS.Add(key)
}

// networkserviceChannelEnqueue will add an object 'obj' into the workqueue. The object being added
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
	// Add the item to the queue
	queueNSC.Add(key)
}

// networkserviceEndpointEnqueue will add an object 'obj' into the workqueue. The object being added
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
	// Add the item to the queue
	queueNSE.Add(key)
}
