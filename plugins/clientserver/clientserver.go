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

package clientserver

import (
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
}

// Init initializes ObjectStore
func (p *Plugin) Init() error {
	p.Log.SetLevel(logging.DebugLevel)
	p.pluginStopCh = make(chan struct{})

	p.Log.Info("><SB> Clinet Server plugin has been initialized.")
	return nil
}

// AfterInit is called for post init processing
func (p *Plugin) AfterInit() error {
	p.Log.Info("AfterInit")
	var err error
	go func() {
		err := ObjectStoreCommunicator(p)
		p.Log.Errorf("ClientServer.AfterInit failed to start ObjectStoreCommunicator with error: %+v", err)
	}()
	return err
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	p.Log.Info("Close")

	return nil
}

// ObjectStoreCommunicator is used to communicate with ObjectStore
func ObjectStoreCommunicator(p *Plugin) error {

	return nil
}
