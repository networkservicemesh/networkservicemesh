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

	"github.com/ligato/networkservicemesh/utils/idempotent"

	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Plugin is the base plugin object for this CRD handler
type Plugin struct {
	idempotent.Impl
	Deps

	pluginStopCh    chan struct{}
	k8sClientConfig *rest.Config
	k8sClientset    *kubernetes.Clientset
	objectStore     objectstore.Interface
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	Name       string
	Log        logger.FieldLoggerPlugin
	Cmd        *cobra.Command
	KubeConfig string // Fetch kubeconfig file from --kube
}

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (p *Plugin) Init() error {
	return p.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	err := p.Log.Init()
	if err != nil {
		return err
	}
	p.pluginStopCh = make(chan struct{})

	p.Log.WithField("kubeconfig", p.KubeConfig).Info("Loading kubernetes client config")
	p.k8sClientConfig, err = clientcmd.BuildConfigFromFlags("", p.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client config: %s", err)
	}

	p.k8sClientset, err = kubernetes.NewForConfig(p.k8sClientConfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %s", err)
	}

	return p.afterInit()
}

// afterInit is called for post init processing
func (p *Plugin) afterInit() error {
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
	return p.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	p.Log.Info("Close")

	return p.Log.Close()
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
