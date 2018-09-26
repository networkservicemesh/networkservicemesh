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
	"strings"
	"time"

	dataplaneutils "github.com/ligato/networkservicemesh/pkg/dataplane/utils"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/testdataplane"
	finalizerutils "github.com/ligato/networkservicemesh/plugins/finalizer/utils"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CleanUp calls for nse or client clean up depending on a pod type
func (p *Plugin) CleanUp(pod *v1.Pod) {
	// Check pod's application label, only pods with application label set will be cleaned up
	label, ok := pod.ObjectMeta.Labels[nsmAppLabel]
	if !ok {
		p.Log.Errorf("pod %s/%s is missing %s label, stopping cleanup", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, nsmAppLabel)
		return
	}

	p.Log.Infof("Initiating cleanup for pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)

	switch label {
	case nsmAppNSE:
		// Start go routine to cleanup nse pod and return control. Go routine will periodicaly attemp to cleanup
		// all finalizers from nse pod, once it is completed, deleting operation will be unblocked.
		go cleanUpNSE(p, pod)
		return
	case nsmAppClient:
		// Start go routine to cleanup nsm pod and return control. Go routine will periodicaly attemp to cleanup
		// all finalizers from nsm pod, once it is completed, deleting operation will be unblocked.
		go cleanUpNSMClient(p, pod)
		return
	default:
		p.Log.Errorf("found nsm pod %s/%s with unknown app label: %s : %s, stopping cleanup", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, nsmAppLabel, label)
		return
	}
}

func cleanUpNSE(plugin *Plugin, pod *v1.Pod) {
	plugin.Log.Infof("attempting cleanup nse pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			// Since NSE gets deleted, cleanUpNSE will remove all NetwordServiceEndpoint objected advertied by this NSE
			endpointsList, err := plugin.nsmClient.NetworkserviceV1().NetworkServiceEndpoints(plugin.namespace).List(metav1.ListOptions{})
			if err != nil {
				plugin.Log.Errorf("fail to list Network Service Endpoints while deleting NSE pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
				continue
			}
			found := false
			if len(endpointsList.Items) != 0 {
				for _, endpoint := range endpointsList.Items {
					if strings.Compare(endpoint.Spec.NseProviderName, string(pod.ObjectMeta.UID)) == 0 {
						// Removing NetworkServiceEndpoint since it was provided by NSE pod about to be deleted
						plugin.nsmClient.NetworkserviceV1().NetworkServiceEndpoints(plugin.namespace).Delete(endpoint.ObjectMeta.Name, &metav1.DeleteOptions{})
						found = true
					}
				}
				if found {
					// All nse advertised endpoints were clreared and now Endpoint finalizer can be safely removed
					if err := finalizerutils.RemovePodFinalizer(plugin.k8sClient, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, EndpointFinalizer); err != nil {
						plugin.Log.Warnf("fail to remove endpoint finalizer from NSE pod %s/%s with error: %+v", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
					} else {
						plugin.Log.Infof("successfully removed endpoint finalizer from NSE pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
					}
				}
			}
			// Check if NSE does not have any active clients using its advertised Endpoints.
			inUse, err := checkForInUse(plugin.k8sClient, pod, plugin.namespace)
			if err != nil {
				plugin.Log.Errorf("failure to checkForInUse for nse pod %s/%s with error: %+v", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
				continue
			}
			if inUse {
				plugin.Log.Warnf("nse pod %s/%s is still being used by nsm clients", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
				continue
			}
			// All clear proceed for nse pod dataplane cleanup
			if err := dataplaneutils.CleanupPodDataplane(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, testdataplane.NSMPodType_NSE); err != nil {
				// NSE pod is about to be deleted as such there is no reason to fail, even if
				// dataplane cleanup failed, simply print an error message is sufficient
				plugin.Log.Errorf("failed to clean up pod %s/%s dataplane with error: %+v, please review dataplane controller log if further debugging is required",
					pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
			}
			plugin.Log.Infof("nse pod %s/%s clean up completed successfully", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
			return
		}
	}
}

func cleanUpNSMClient(plugin *Plugin, pod *v1.Pod) {
	plugin.Log.Infof("attempting cleanup nsm pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			if err := dataplaneutils.CleanupPodDataplane(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, testdataplane.NSMPodType_NSMCLIENT); err != nil {
				// NSM pod is about to be deleted as such there is no reason to fail, even if
				// dataplane cleanup failed, simply print an error message is sufficient
				plugin.Log.Errorf("failed to clean up pod %s/%s dataplane with error: %+v, please review dataplane controller log if further debugging is required",
					pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
			}
			plugin.Log.Infof("successfully removed dataplane from NSM pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
			if err := finalizerutils.RemovePodFinalizer(plugin.k8sClient, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, NSMFinalizer); err != nil {
				plugin.Log.Warnf("fail to remove finalizers from NSM pod %s/%s with error: %+v", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
				continue
			} else {
				plugin.Log.Infof("successfully removed finalizers from NSM pod %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
			}
			return
		}
	}
}

func checkForInUse(k8s *kubernetes.Clientset, pod *v1.Pod, podNamespace string) (bool, error) {
	var inUse bool
	finalizers := pod.GetFinalizers()
	nsmClients := []string{}
	// Bulding list of nse pod clients, they exists as nse finalizers entries
	for _, fn := range finalizers {
		if strings.HasSuffix(fn, NSEFinalizerSuffix) {
			nsmClients = append(nsmClients, strings.Split(fn, NSEFinalizerSuffix)[0])
		}
	}
	// Check if any of nsm client pods are still alive, if it is the case nse pod still cannot be deleted
	// and inUse true is returned, otherwise if nsm client pod is not alive, its name is removed from the list of finalizers.
	for _, nsm := range nsmClients {
		_, err := k8s.CoreV1().Pods(podNamespace).Get(nsm, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// nsm client can be removed from nse pod finalizer list
			finalizerutils.RemovePodFinalizer(k8s, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, nsm+NSEFinalizerSuffix)
		}
		if err == nil {
			// nsm client pod is still alive
			inUse = true
		}
	}

	return inUse, nil
}
