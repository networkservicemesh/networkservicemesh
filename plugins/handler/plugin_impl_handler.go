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

package handler

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Plugin is the base plugin object for this CRD handler
type Plugin struct {
	Deps

	pluginStopCh    chan struct{}
	k8sClientConfig *rest.Config
	k8sClientset    *kubernetes.Clientset
	objectStore     objectstore.Interface
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	local.PluginInfraDeps
	KubeConfig config.PluginConfig
}

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (p *Plugin) Init() error {
	var err error
	p.Log.SetLevel(logging.DebugLevel)
	p.pluginStopCh = make(chan struct{})

	kubeconfig := p.KubeConfig.GetConfigName()
	p.Log.WithField("kubeconfig", kubeconfig).Info("Loading kubernetes client config")
	p.k8sClientConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("Failed to build kubernetes client config: %s", err)
	}

	p.k8sClientset, err = kubernetes.NewForConfig(p.k8sClientConfig)
	if err != nil {
		return fmt.Errorf("Failed to build kubernetes client: %s", err)
	}

	return nil
}

// AfterInit is called for post init processing
func (p *Plugin) AfterInit() error {
	p.Log.Info("AfterInit")

	ticker := time.NewTicker(objectstore.ObjectStoreReadyInterval)
	timeout := time.After(objectstore.ObjectStoreReadyTimeout)
	defer ticker.Stop()
	// Wait for objectstore to initialize
	ready := false
	for !ready {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for ObjectStore")
		case <-ticker.C:
			if p.objectStore = objectstore.SharedPlugin(); p.objectStore != nil {
				ready = true
				ticker.Stop()
				p.Log.Info("ObjectStore is ready, starting Consumer")
			} else {
				p.Log.Info("ObjectStore is not ready, waiting")
			}
		}
	}

	return nil
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	p.Log.Info("Close")

	return nil
}

// ObjectCreated is called when an object is created
func (p *Plugin) ObjectCreated(obj interface{}) {
	p.Log.Infof("LogCrdHandler.ObjectCreated: ", reflect.TypeOf(obj), obj)
	p.objectStore.ObjectCreated(obj)
}

// ObjectDeleted is called when an object is deleted
func (p *Plugin) ObjectDeleted(obj interface{}) {
	p.Log.Infof("LogCrdHandler.ObjectDeleted: ", reflect.TypeOf(obj), obj)
	p.objectStore.ObjectDeleted(obj)
}

// ObjectUpdated is called when an object is updated
func (p *Plugin) ObjectUpdated(old, cur interface{}) {
	p.Log.Infof("LogCrdHandler.ObjectUpdated: ", reflect.TypeOf(old), reflect.TypeOf(cur), old, cur)
	p.objectStore.ObjectDeleted(old)
	p.objectStore.ObjectCreated(cur)
}
