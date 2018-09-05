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

package interrupthandler_test

import (
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/ligato/networkservicemesh/plugins/interrupthandler"

	"github.com/ligato/networkservicemesh/utils/idempotent"
)

type Plugin struct {
	idempotent.Impl
}

func (p *Plugin) Init() error {
	return p.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	return nil
}

func (p *Plugin) Close() error {
	return p.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	return nil
}

func TestWrap(t *testing.T) {
	plugin := &Plugin{}
	interupt := interrupthandler.Wrap(plugin)
	interupt.Init()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done() // Release the wg when we are done
		interupt.Wait()
	}()

	// Send the signal
	syscall.Kill(os.Getpid(), syscall.SIGINT)
	wg.Wait()
}
