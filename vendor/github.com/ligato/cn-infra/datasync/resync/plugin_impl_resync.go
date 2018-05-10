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

package resync

import (
	"sync"
	"time"

	"github.com/ligato/cn-infra/flavors/local"
)

const (
	singleResyncTimeout = time.Second * 5
)

// Plugin implements Plugin interface, therefore it can be loaded with other plugins.
type Plugin struct {
	Deps

	regOrder      []string
	registrations map[string]Registration
	access        sync.Mutex
}

// Deps groups dependencies injected into the plugin so that they are
// logically separated from other plugin fields.
type Deps struct {
	local.PluginLogDeps // inject
}

// Init initializes variables.
func (plugin *Plugin) Init() (err error) {
	plugin.registrations = make(map[string]Registration)

	//plugin.waingForResync = make(map[core.PluginName]*PluginEvent)
	//plugin.waingForResyncChan = make(chan *PluginEvent)
	//go plugin.watchWaingForResync()

	return nil
}

// AfterInit method starts the resync.
func (plugin *Plugin) AfterInit() (err error) {
	plugin.startResync()

	return nil
}

// Close TODO set flag that ignore errors => not start Resync while agent is stopping
// TODO kill existing Resync timeout while agent is stopping
func (plugin *Plugin) Close() error {
	//TODO close error report channel

	plugin.access.Lock()
	defer plugin.access.Unlock()

	plugin.registrations = make(map[string]Registration)

	return nil
}

// Register function is supposed to be called in Init() by all VPP Agent plugins.
// The plugins are supposed to load current state of their objects when newResync() is called.
// The actual CreateNewObjects(), DeleteObsoleteObjects() and ModifyExistingObjects() will be orchestrated
// to ensure their proper order. If an error occurs during Resync, then new Resync is planned.
func (plugin *Plugin) Register(resyncName string) Registration {
	plugin.access.Lock()
	defer plugin.access.Unlock()

	if _, found := plugin.registrations[resyncName]; found {
		plugin.Log.WithField("resyncName", resyncName).
			Panic("You are trying to register same resync twice")
		return nil
	}
	// ensure that resync is triggered in the same order as the plugins were registered
	plugin.regOrder = append(plugin.regOrder, resyncName)

	reg := NewRegistration(resyncName, make(chan StatusEvent, 0)) /*Zero to have back pressure*/
	plugin.registrations[resyncName] = reg

	return reg
}

// Call callback on plugins to create/delete/modify objects.
func (plugin *Plugin) startResync() {
	plugin.Log.Info("Resync order", plugin.regOrder)

	startResyncTime := time.Now()

	for _, regName := range plugin.regOrder {
		if reg, found := plugin.registrations[regName]; found {
			startPartTime := time.Now()

			plugin.startSingleResync(regName, reg)

			took := time.Since(startPartTime)
			plugin.Log.WithField("durationInNs", took.Nanoseconds()).
				Infof("Resync of %v took %v", regName, took)
		}
	}

	took := time.Since(startResyncTime)
	plugin.Log.WithField("durationInNs", took.Nanoseconds()).Info("Resync took ", took)

	// TODO check if there ReportError (if not than report) if error occurred even during Resync
}
func (plugin *Plugin) startSingleResync(resyncName string, reg Registration) {
	started := newStatusEvent(Started)
	reg.StatusChan() <- started

	select {
	case <-started.ReceiveAck():
	case <-time.After(singleResyncTimeout):
		plugin.Log.WithField("regName", resyncName).Warn("Timeout of ACK")
	}
}
