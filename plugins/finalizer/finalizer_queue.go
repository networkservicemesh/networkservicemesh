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
	"strings"

	dataplaneutils "github.com/ligato/networkservicemesh/pkg/dataplane/utils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/testdataplane"
	finalizerutils "github.com/ligato/networkservicemesh/plugins/finalizer/utils"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	plugin.Log.Infof("found pod %s/%s with label: %s : %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, nsmAppLabel, label)

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
	// Since NSE gets deleted, cleanUpNSE will remove all NetwordServiceEndpoint objected advertied by this NSE
	endpointsList, err := plugin.nsmClient.NetworkserviceV1().NetworkServiceEndpoints(plugin.namespace).List(metav1.ListOptions{})
	if err == nil {
		if len(endpointsList.Items) != 0 {
			for _, endpoint := range endpointsList.Items {
				if strings.Compare(endpoint.Spec.NseProviderName, string(pod.ObjectMeta.UID)) == 0 {
					// Removing NetworkServiceEndpoint since it was provided by NSE pod about to be deleted
					plugin.nsmClient.NetworkserviceV1().NetworkServiceEndpoints(plugin.namespace).Delete(endpoint.ObjectMeta.Name, &metav1.DeleteOptions{})
				}
			}
		} else {
			plugin.Log.Warnf("NSE pod %s/%s has not been providing any Network Service Endpoints", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		}
	} else {
		plugin.Log.Errorf("fail to list Network Service Endpoints while deleting NSE pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	}
	if err := dataplaneutils.CleanupPodDataplane(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, testdataplane.NSMPodType_NSE); err != nil {
		// NSE pod is about to be deleted as such there is no reason to fail, even if
		// dataplane cleanup failed, simply print an error message is sufficient
		plugin.Log.Errorf("failed to clean up pod %s/%s dataplane with error: %+v, please review dataplane controller log if further debugging is required",
			pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
	}
	if err := finalizerutils.RemovePodFinalizer(plugin.k8sClient, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace); err != nil {
		plugin.Log.Warnf("fail to remove finalizers from NSM pod %s/%s with error: %+v", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
	} else {
		plugin.Log.Infof("successfully removed finalizers from NSM pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	}

	return nil
}

func cleanUpNSMClient(plugin *Plugin, pod *v1.Pod) error {
	plugin.Log.Infof("cleanup requested for NSM Client pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)

	if err := dataplaneutils.CleanupPodDataplane(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, testdataplane.NSMPodType_NSMCLIENT); err != nil {
		// NSM pod is about to be deleted as such there is no reason to fail, even if
		// dataplane cleanup failed, simply print an error message is sufficient
		plugin.Log.Errorf("failed to clean up pod %s/%s dataplane with error: %+v, please review dataplane controller log if further debugging is required",
			pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
	}
	plugin.Log.Infof("successfully removed dataplane from NSM pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	if err := finalizerutils.RemovePodFinalizer(plugin.k8sClient, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace); err != nil {
		plugin.Log.Warnf("fail to remove finalizers from NSM pod %s/%s with error: %+v", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
	} else {
		plugin.Log.Infof("successfully removed finalizers from NSM pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	}

	return nil
}
