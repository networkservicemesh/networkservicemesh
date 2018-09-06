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

package interrupthandler

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ligato/networkservicemesh/utils/idempotent"
)

// Plugin implements PluginAPI
type Plugin struct {
	idempotent.Impl
	Deps
	doneCh chan struct{}
}

// Init Plugin
func (p *Plugin) Init() error {
	return p.Impl.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	// Always set up for the signal before you do the Init()
	// So that if its caught while you are running Init() you still
	// catch it
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)
	p.doneCh = make(chan struct{})
	err := p.Deps.Wrap.Init()
	if err == nil {
		go func() {
			select {
			case <-sigChan:
			}
			p.Close()
		}()
	}
	return err
}

// Close Plugin
func (p *Plugin) Close() error {
	return p.Impl.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	err := p.Wrap.Close()
	close(p.doneCh)
	return err
}

// After returns a readonly channel that is Closed with the Plug in is Closed
func (p *Plugin) After() <-chan struct{} {
	return p.doneCh
}

// Wait waits until the Plugin is Closed
func (p *Plugin) Wait() {
	<-p.After()
}
