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

package nsmcommand

import (
	"github.com/ligato/networkservicemesh/utils/helper"
	"github.com/ligato/networkservicemesh/utils/idempotent"
)

// Plugin for nsmcommand
type Plugin struct {
	idempotent.Impl
	Deps
}

// Init Plugin
func (p *Plugin) Init() error {
	return p.Impl.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	err := p.Cmd.Execute()
	if err != nil {
		p.Log.Errorf("Initializing NSMServer failed with error: %s", err)
		return err
	}
	err = helper.InitDeps(p)
	if err != nil {
		p.Log.Errorf("Initializing NSMServer failed with error: %s", err)
		return err
	}
	return nil
}

// Close Plugin
func (p *Plugin) Close() error {
	return p.Impl.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	return helper.CloseDeps(p)
}
