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
	"reflect"
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// QueueRetryCount is the max number of times to retry processing a failed item
// from the workqueue.
const QueueRetryCount = 5

const (
	createOp = iota
	deleteOp
	updateOp
)

type objectMessage struct {
	operation int
	key       string
	obj       interface{}
}

var (
	// These are queues of resources to be processed. They each performs
	// exponential backoff rate limiting, with a minimum retry period of 5
	// seconds and a maximum of 1 minute.
	queueNS  = workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(time.Second*5, time.Minute))
	queueNSC = workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(time.Second*5, time.Minute))
	queueNSE = workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(time.Second*5, time.Minute))
)

// This file contains the work queue backends for each of the CRDs we create in the
// plugin AfterInit() call.

func workforever(plugin *Plugin, queue workqueue.RateLimitingInterface, informer cache.SharedIndexInformer, stopCH chan struct{}) {
	for {
		// We read a message off the queue ...
		message, shutdown := queue.Get()

		// If the queue has been shut down, we should exit the work queue here
		if shutdown {
			plugin.Log.Error("shutdown signaled, closing stopChNS")
			close(stopCH)
			return
		}

		// Convert the queue item into a string. If it's not a string, we'll
		// simply discard it as invalid data and log a message.
		var strKey string
		strKey = message.(objectMessage).key

		// We define a function here to process a queue item, so that we can
		// use 'defer' to make sure the message is marked as Done on the queue
		func(key string) {
			defer queue.Done(message)

			// Attempt to split the 'key' into namespace and object name
			namespace, name, err := cache.SplitMetaNamespaceKey(strKey)

			if err != nil {
				// This is a soft-error
				plugin.Log.Errorf("Error splitting meta namespace key into parts: %s", err.Error())
				queue.Forget(message)
				return
			}

			plugin.Log.Infof("Read item '%s/%s' off workqueue. Processing...", namespace, name)

			// Retrieve the latest version in the cache of this NetworkService. By using
			// GetByKey() we are able to determine if the item exists, and thus if it was
			// added or deleted from the queue, and process appropriately
			item, _, err := informer.GetIndexer().GetByKey(strKey)

			if err != nil {
				if queue.NumRequeues(key) < QueueRetryCount {
					plugin.Log.Errorf("Requeueing after error processing item with key %s, error %v", key, err)
					queue.AddRateLimited(message)
					return
				}

				plugin.Log.Errorf("Failed processing item with key %s, error %v, no more retries", key, err)
				queue.Forget(message)
				return
			}

			plugin.Log.Infof("Found object of type: %T", reflect.TypeOf(message.(objectMessage).obj))
			// Check if this is a create or delete operation
			switch message.(objectMessage).operation {
			case createOp:
				// Verify and log if the informer cached version of the object is different than the
				// copy we made
				if !reflect.DeepEqual(message.(objectMessage).obj, item) {
					plugin.Log.Errorf("Informer cached version of object (%s/%s) different than worker queue version", namespace, name)
				}
				plugin.Log.Infof("Got most up to date version of '%s/%s'. Syncing...", namespace, name)
				plugin.objectStore.ObjectCreated(message.(objectMessage).obj)
			case deleteOp:
				plugin.Log.Infof("Got most up to date version of '%s/%s'. Syncing...", namespace, name)
				plugin.objectStore.ObjectDeleted(message.(objectMessage).obj)
			}

			// As we managed to process this successfully, we can forget it
			// from the work queue altogether.
			queue.Forget(message)
		}(strKey)
	}
}
