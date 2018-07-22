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
	"reflect"

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

func workforever(plugin *Plugin, queue workqueue.RateLimitingInterface, informer cache.SharedIndexInformer, stopCH chan struct{}) {
	for {
		message, shutdown := queue.Get()

		// If the queue has been shut down, we should exit the work queue here
		if shutdown {
			plugin.Log.Error("shutdown signaled, closing stopChNS")
			close(stopCH)
			return
		}

		var strKey string
		strKey = message.(objectMessage).key

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

			plugin.Log.Infof("Found object of type: %s", reflect.TypeOf(message.(objectMessage).obj))
			// Check if this is a create or delete operation
			switch message.(objectMessage).operation {
			case deleteOp:
				plugin.Log.Infof("Got most up to date version of '%s/%s'. Syncing...", namespace, name)

				plugin.Log.Infof("><SB> Got delete event for object type %+s", reflect.TypeOf(message.(objectMessage).obj))
				plugin.ObjectStore.ObjectDeleted(message.(objectMessage).obj)
			}

			// As we managed to process this successfully, we can forget it
			// from the work queue altogether.
			queue.Forget(message)
		}(strKey)
	}
}
