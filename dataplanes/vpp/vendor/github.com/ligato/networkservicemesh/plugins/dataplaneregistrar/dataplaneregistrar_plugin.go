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

// //go:generate protoc -I ./model/pod --go_out=plugins=grpc:./model/pod ./model/pod/pod.proto

package dataplaneregistrar

import (
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	"github.com/ligato/networkservicemesh/utils/registry"
)

// Plugin groups together resources and functions required for nsm server to run
type Plugin struct {
	Deps
	pluginStopCh chan bool
	idempotent.Impl
}

// Deps defines dependencies of netmesh plugin.
type Deps struct {
	Name        string
	Log         logger.FieldLoggerPlugin
	ObjectStore objectstore.PluginAPI
}

// Init initializes ObjectStore plugin
func (p *Plugin) Init() error {
	return p.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	p.pluginStopCh = make(chan bool, 1)
	err := deptools.Init(p)
	if err != nil {
		return err
	}
	return NewDataplaneRegistrarServer(p)
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	return p.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	p.Log.Info("Close")
	p.pluginStopCh <- true
	registry.Shared().Delete(p)
	return deptools.Close(p)
}
