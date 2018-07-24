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
	"reflect"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func workforever(plugin *Plugin, queue workqueue.RateLimitingInterface, informer cache.SharedIndexInformer, stopCH chan struct{}) {
	for {
		obj, shutdown := queue.Get()

		// If the queue has been shut down, we should exit the work queue here
		if shutdown {
			plugin.Log.Error("shutdown signaled, closing stopChNS")
			close(stopCH)
			return
		}

		func(obj interface{}) {
			defer queue.Done(obj)
			pod, ok := obj.(*v1.Pod)
			if !ok {
				plugin.Log.Errorf("Unexpected object type %s", reflect.TypeOf(obj))
				queue.Forget(obj)
				return
			}

			if err := cleanUp(plugin, pod); err != nil {
				plugin.Log.Errorf("object clean up failed with error: %+v", err)
			} else {
				plugin.Log.Infof("object clean up successful %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
			}
			queue.Forget(obj)
		}(obj)
	}
}

func cleanUp(plugin *Plugin, pod *v1.Pod) error {
	var err error
	label, ok := pod.ObjectMeta.Labels[nsmAppLabel]
	if !ok {
		return fmt.Errorf("pod %s/%s is missing %s label, stopping cleanup", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, nsmAppLabel)
	}
	plugin.Log.Infof("found nsm pod %s/%s with label: %s : %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, nsmAppLabel, label)

	switch label {
	case nsmAppNSE:
		err = cleanUpNSE(plugin, pod)
	case nsmAppClient:
		err = cleanUpNSMClient(plugin, pod)
	default:
		return fmt.Errorf("found nsm pod %s/%s with unknown app label: %s : %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, nsmAppLabel, label)
	}
	return err
}

func cleanUpNSE(plugin *Plugin, pod *v1.Pod) error {
	plugin.Log.Infof("cleanup requested for NSE pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)

	// Step 1 getting a slice of channels advertised by about to be deleted pod
	channels := plugin.ObjectStore.GetChannelsByNSEServerProvider(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
	if channels == nil {
		plugin.Log.Infof("no advertised channels found for NSE pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		return nil
	}
	// Step 2 range through received list of channels and for each found NetworkService, remove the channel
	// from NetworkService object.
	plugin.Log.Infof("found %d advertised channels found for NSE pod %s/%s", len(channels), pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	for _, ch := range channels {
		plugin.Log.Infof("channel %s/%s was used by netowrk service %s, deleting it...", ch.Metadata.Namespace, ch.Metadata.Name, ch.NetworkServiceName)
		if err := plugin.ObjectStore.DeleteChannelFromNetworkService(ch.NetworkServiceName, ch.Metadata.Namespace, ch); err != nil {
			plugin.Log.Errorf("failed channel %s/%s from netowrk service %s with error: %+v", ch.Metadata.Namespace, ch.Metadata.Name, ch.NetworkServiceName, err)
			return err
		}
	}
	// Step 3 last step is to remove from Channels map all channels advertised by the pod
	plugin.ObjectStore.DeleteNSE(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
	plugin.Log.Infof("all channels advertised by NSE %s/%s were deleted", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)

	// Step 4 Clean up dataplane

	return nil
}

func cleanUpNSMClient(plugin *Plugin, pod *v1.Pod) error {
	plugin.Log.Infof("cleanup requested for NSM Client pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	return nil
}
