// Copyright (c) 2017 Cisco and/or its affiliates.
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

package kvdbsync

import (
	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/cn-infra/datasync/resync"
	"github.com/ligato/cn-infra/datasync/syncbase"
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/servicelabel"
)

// Plugin dbsync implements synchronization between local memory and db.
// Other plugins can be notified when DB changes occur or resync is needed.
// This plugin reads/pulls the data from db when resync is needed.
type Plugin struct {
	Deps // inject

	adapter  *watcher
	registry *syncbase.Registry
}

type infraDeps interface {
	// InfraDeps for getting PlugginInfraDeps instance (logger, config, plugin name, statuscheck)
	InfraDeps(pluginName string, opts ...local.InfraDepsOpts) *local.PluginInfraDeps
}

// OfDifferentAgent allows accessing DB of a different agent (with a particular microservice label).
// This method is a shortcut to simplify creating new instance of a plugin
// that is supposed to watch different agent DB.
// Method intentionally copies instance of a plugin (assuming it has set all dependencies)
// and sets microservice label.
func (plugin /*intentionally without pointer receiver*/ Plugin) OfDifferentAgent(
	microserviceLabel string, infraDeps infraDeps) *Plugin {

	// plugin name suffixed by micorservice label
	plugin.Deps.PluginInfraDeps = *infraDeps.InfraDeps(string(
		plugin.Deps.PluginInfraDeps.PluginName) + "-" + microserviceLabel)

	// this is important - here comes microservice label of different agent
	plugin.Deps.PluginInfraDeps.ServiceLabel = servicelabel.OfDifferentAgent(microserviceLabel)
	return &plugin // copy (no pointer receiver)
}

// Deps groups dependencies injected into the plugin so that they are
// logically separated from other plugin fields.
type Deps struct {
	local.PluginInfraDeps                      // inject
	ResyncOrch            resync.Subscriber    // inject
	KvPlugin              keyval.KvProtoPlugin // inject
}

// Init only initializes plugin.registry.
func (plugin *Plugin) Init() error {
	plugin.registry = syncbase.NewRegistry()

	return nil
}

// AfterInit uses provided connection to build new transport watcher.
//
// Plugin.registry subscriptions (registered by Watch method) are used for resync.
// Resync is called only if ResyncOrch was injected (i.e. is not nil).
// The order of plugins in flavor is not important to resync
// since Watch() is called in Plugin.Init() and Resync.Register()
// is called in Plugin.AfterInit().
func (plugin *Plugin) AfterInit() error {
	if plugin.KvPlugin != nil && !plugin.KvPlugin.Disabled() {
		db := plugin.KvPlugin.NewBroker(plugin.ServiceLabel.GetAgentPrefix())
		dbW := plugin.KvPlugin.NewWatcher(plugin.ServiceLabel.GetAgentPrefix())
		plugin.adapter = &watcher{db, dbW, plugin.registry}

		if plugin.ResyncOrch != nil {
			for resyncName, sub := range plugin.registry.Subscriptions() {
				resyncReg := plugin.ResyncOrch.Register(resyncName)
				_, err := watchAndResyncBrokerKeys(resyncReg, sub.ChangeChan, sub.ResyncChan, sub.CloseChan,
					plugin.adapter, sub.KeyPrefixes...)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Watch adds entry to the plugin.registry. By doing this, other plugins will receive notifications
// about data changes and data resynchronization.
//
// This method is supposed to be called in Plugin.Init().
// Calling this method later than kvdbsync.Plugin.AfterInit() will have no effect
// (no notifications will be received).
func (plugin *Plugin) Watch(resyncName string, changeChan chan datasync.ChangeEvent,
	resyncChan chan datasync.ResyncEvent, keyPrefixes ...string) (datasync.WatchRegistration, error) {

	return plugin.registry.Watch(resyncName, changeChan, resyncChan, keyPrefixes...)
}

// Put propagates this call to a particular kvdb.Plugin unless the kvdb.Plugin is Disabled().
//
// This method is supposed to be called in Plugin.AfterInit() or later (even from different go routine).
func (plugin *Plugin) Put(key string, data proto.Message, opts ...datasync.PutOption) error {
	if plugin.KvPlugin.Disabled() {
		return nil
	}

	if plugin.adapter != nil {
		return plugin.adapter.db.Put(key, data, opts...)
	}

	return errors.New("Transport adapter is not ready yet. (Probably called before AfterInit)")
}

// Delete propagates this call to a particular kvdb.Plugin unless the kvdb.Plugin is Disabled().
//
// This method is supposed to be called in Plugin.AfterInit() or later (even from different go routine).
func (plugin *Plugin) Delete(key string, opts ...datasync.DelOption) (existed bool, err error) {
	if plugin.KvPlugin.Disabled() {
		return false, nil
	}

	if plugin.adapter != nil {
		return plugin.adapter.db.Delete(key, opts...)
	}

	return false, errors.New("Transport adapter is not ready yet. (Probably called before AfterInit)")
}

// Close resources.
func (plugin *Plugin) Close() error {
	return nil
}

// String returns Deps.PluginName if set, "kvdbsync" otherwise.
func (plugin *Plugin) String() string {
	if len(plugin.PluginName) == 0 {
		return "kvdbsync"
	}
	return string(plugin.PluginName)
}
