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

package main

import (
	"time"

	"github.com/ligato/cn-infra/db/keyval/etcd"
	"github.com/ligato/cn-infra/health/statuscheck"

	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/datasync/kvdbsync"
	"github.com/ligato/cn-infra/datasync/resync"
	"github.com/ligato/cn-infra/flavors/connectors"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/utils/safeclose"
)

// *************************************************************************
// This example demonstrates the usage of StatusReader API
// ETCD plugin is monitored by status check plugin.
// ExamplePlugin periodically prints the status.
// ************************************************************************/

func main() {
	// Init close channel used to stop the example.
	exampleFinished := make(chan struct{}, 1)

	// Start Agent with ExamplePlugin, ETCDPlugin & FlavorLocal (reused cn-infra plugins).
	agent := local.NewAgent(local.WithPlugins(func(flavor *local.FlavorLocal) []*core.NamedPlugin {
		etcdPlug := &etcd.Plugin{}
		etcdDataSync := &kvdbsync.Plugin{}
		resyncOrch := &resync.Plugin{}

		etcdPlug.Deps.PluginInfraDeps = *flavor.InfraDeps("etcd", local.WithConf())
		resyncOrch.Deps.PluginLogDeps = *flavor.LogDeps("etcd-resync")
		connectors.InjectKVDBSync(etcdDataSync, etcdPlug, etcdPlug.PluginName, flavor, resyncOrch)

		examplePlug := &ExamplePlugin{closeChannel: exampleFinished}
		examplePlug.PluginInfraDeps = *flavor.InfraDeps("statuscheck-example")
		examplePlug.StatusMonitor = &flavor.StatusCheck // Inject status check

		return []*core.NamedPlugin{
			{etcdPlug.PluginName, etcdPlug},
			{etcdDataSync.PluginName, etcdDataSync},
			{resyncOrch.PluginName, resyncOrch},
			{examplePlug.PluginName, examplePlug}}
	}))
	core.EventLoopWithInterrupt(agent, nil)
}

// ExamplePlugin demonstrates the usage of datasync API.
type ExamplePlugin struct {
	local.PluginInfraDeps // injected
	StatusMonitor         statuscheck.StatusReader

	// Fields below are used to properly finish the example.
	closeChannel chan struct{}
}

// Init starts the consumer.
func (plugin *ExamplePlugin) Init() error {
	return nil
}

// AfterInit starts the publisher and prepares for the shutdown.
func (plugin *ExamplePlugin) AfterInit() error {

	go plugin.checkStatus(plugin.closeChannel)

	return nil
}

// checkStatus periodically prints status of plugins that publish their state
// to status check plugin
func (plugin *ExamplePlugin) checkStatus(closeCh chan struct{}) {
	for {
		select {
		case <-closeCh:
			plugin.Log.Info("Closing")
			return
		case <-time.After(1 * time.Second):
			status := plugin.StatusMonitor.GetAllPluginStatus()
			for k, v := range status {
				plugin.Log.Infof("Status[%v] = %v", k, v)
			}
		}
	}
}

// Close shutdowns the consumer and channels used to propagate data resync and data change events.
func (plugin *ExamplePlugin) Close() error {
	return safeclose.Close(plugin.closeChannel)
}
