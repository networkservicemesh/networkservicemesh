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
		return err
	}
	// TODO: Figure out correct order of initialization
	err = p.Deps.Log.Init()
	if err != nil {
		return err
	}
	err = p.Deps.ObjectStore.Init()
	if err != nil {
		p.Log.Errorf("Initializing ObjectStore failed with error: %s", err)
		return err
	}
	err = p.Deps.CRD.Init()
	if err != nil {
		p.Log.Errorf("Initializing CRD failed with error: %s", err)
		return err
	}
	err = p.Deps.NSMServer.Init()
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
	// TODO: Figure out correct order of initialization
	err := p.Deps.Log.Close()
	if err != nil {
		return err
	}

	err = p.Deps.NSMServer.Close()
	if err != nil {
		p.Log.Errorf("Closing NSMServer failed with error: %s", err)
		return err
	}

	err = p.Deps.CRD.Close()
	if err != nil {
		p.Log.Errorf("Closing CRD failed with error: %s", err)
		return err
	}

	err = p.Deps.ObjectStore.Close()
	if err != nil {
		p.Log.Errorf("Closing ObjectStore failed with error: %s", err)
		return err
	}
	return nil
}
