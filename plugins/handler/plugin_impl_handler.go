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
	"reflect"

	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	"github.com/ligato/networkservicemesh/utils/helper/plugintools"
	"github.com/ligato/networkservicemesh/utils/registry"

	"github.com/ligato/networkservicemesh/utils/idempotent"

	"github.com/ligato/networkservicemesh/plugins/k8sclient"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"github.com/spf13/cobra"
)

// Plugin is the base plugin object for this CRD handler
type Plugin struct {
	idempotent.Impl
	Deps

	pluginStopCh chan struct{}
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	Name        string
	Log         logger.FieldLogger
	Cmd         *cobra.Command
	KubeConfig  string `empty_value_ok:"true"` // Fetch kubeconfig file from --kube-config
	ObjectStore objectstore.Interface
	K8sclient   k8sclient.API
}

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (p *Plugin) Init() error {
	return p.IdempotentInit(plugintools.LoggingInitFunc(p.Log, p, p.init))
}

func (p *Plugin) init() error {
	p.pluginStopCh = make(chan struct{})
	err := deptools.Init(p)
	if err != nil {
		return err
	}

	p.Log.WithField("kubeconfig", p.KubeConfig).Info("Loading kubernetes client config")

	return nil
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	return p.IdempotentClose(plugintools.LoggingCloseFunc(p.Log, p, p.close))
}

func (p *Plugin) close() error {
	p.Log.Info("Close")
	registry.Shared().Delete(p)
	err := deptools.Close(p)
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
