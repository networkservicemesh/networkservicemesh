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

package objectstore

import (
	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/logging"
)

// Plugin is the base plugin object for this CRD handler
type Plugin struct {
	Deps

	pluginStopCh chan struct{}
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	local.PluginInfraDeps
	KubeConfig config.PluginConfig
}

// Init builds K8s client-set based on the supplied kubeconfig and initializes
// all reflectors.
func (p *Plugin) Init() error {
	p.Log.SetLevel(logging.DebugLevel)
	p.pluginStopCh = make(chan struct{})

	return nil
}

// AfterInit is called for post init processing
func (p *Plugin) AfterInit() error {
	p.Log.Info("AfterInit")

	return nil
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	p.Log.Info("Close")

	return nil
}

// ObjectCreated is called when an object is created
func (p *Plugin) ObjectCreated(obj interface{}) {
	p.Log.Infof("LogCrdHandler.ObjectCreated: %s", obj)
}

// ObjectDeleted is called when an object is deleted
func (p *Plugin) ObjectDeleted(obj interface{}) {
	p.Log.Infof("LogCrdHandler.ObjectDeleted: %s", obj)
}

// ObjectUpdated is called when an object is updated
func (p *Plugin) ObjectUpdated(objOld, objNew interface{}) {
	p.Log.Infof("LogCrdHandler.ObjectUpdated: %s", objNew)
}

// meta is used as a key for each NetworkService object
type meta struct {
	name      string
	namespace string
}

// ObjectStore stores information about all objects learned by CRDs controller
// TODO add NetworkServiceEndpoint and NetworkServiceChannel
type ObjectStore struct {
	NetworkServicesStore
}
