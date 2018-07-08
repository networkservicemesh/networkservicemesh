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
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	Name        string
	Log         logger.FieldLoggerPlugin
	Cmd         *cobra.Command
	KubeConfig  string // Fetch kubeconfig file from --kube-config
	ObjectStore objectstore.PluginAPI
}

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (p *Plugin) Init() error {
	return p.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	p.pluginStopCh = make(chan struct{})
	err := p.Log.Init()
	if err != nil {
		return err
	}
	err = p.Deps.ObjectStore.Init()
	if err != nil {
		return err
	}

	p.Log.WithField("kubeconfig", p.KubeConfig).Info("Loading kubernetes client config")
	p.k8sClientConfig, err = clientcmd.BuildConfigFromFlags("", p.KubeConfig)
	if err != nil {
		p.Log.Infof("Failed to build kubernetes client config, will try cluster config.  err: %s", err)
		p.k8sClientConfig, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("Failed to build kubernetes client config from rest.InClusterConfig(): %s", err)
		}
	}

	p.k8sClientset, err = kubernetes.NewForConfig(p.k8sClientConfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes client: %s", err)
	}

	return nil
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	return p.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	p.Log.Info("Close")
	err := p.Log.Close()
	if err != nil {
		return err
	}
	err = p.ObjectStore.Close()
	if err != nil {
		return err
	}

	return err
}

// ObjectCreated is called when an object is created
func (p *Plugin) ObjectCreated(obj interface{}) {
	p.Log.Infof("LogCrdHandler.ObjectCreated: ", reflect.TypeOf(obj), obj)
	p.ObjectStore.ObjectCreated(obj)
}

// ObjectDeleted is called when an object is deleted
func (p *Plugin) ObjectDeleted(obj interface{}) {
	p.Log.Infof("LogCrdHandler.ObjectDeleted: ", reflect.TypeOf(obj), obj)
	p.ObjectStore.ObjectDeleted(obj)
}

// ObjectUpdated is called when an object is updated
func (p *Plugin) ObjectUpdated(old, cur interface{}) {
	p.Log.Infof("LogCrdHandler.ObjectUpdated: ", reflect.TypeOf(old), reflect.TypeOf(cur), old, cur)
	p.ObjectStore.ObjectDeleted(old)
	p.ObjectStore.ObjectCreated(cur)
}
