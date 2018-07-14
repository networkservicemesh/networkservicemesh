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

// Package core manages the lifecycle of all plugins (start, graceful
// shutdown) and defines the core lifecycle SPI. The core lifecycle SPI
// must be implemented by each plugin.

package idempotent

import (
	"errors"
	"sync"
)

const (
	// ReinitErrorStr is the string used for errors returned when you attempt
	// to close an already Closed Plugin
	ReinitErrorStr = "true Close() has already occurred, plugin can no longer be Init() ed"
)

// Impl implements methods for wrapping Init() and Close() such that
// the actual Init() and Close() are idempotent.
type Impl struct {
	refCount      int
	refCountMutex sync.Mutex
	initErr       error
	closeErr      error
}

// IsIdempotent returns true if the object is idempotent
// Its mostly used as a marker to match the IdemPotent interface
func (i *Impl) IsIdempotent() bool {
	return true
}

// IsClosed returns true if an Idempotent plugin is *truely* closed
func (i *Impl) IsClosed() bool {
	i.refCountMutex.Lock()
	defer i.refCountMutex.Unlock()
	return i.refCount < 0
}

// Init should be overriden by a Plugin embedding idempotent.Impl
// When overriding use something like:
// func (p *Plugin) Init() error {
// 	return p.Impl.IdempotentInit(p.init)
// }
// where p.init() is contains the real init code for your plugin
func (i *Impl) Init() error { return nil }

// Close should be overriden
// When overriding use something like:
// func (p *Plugin) Close() error {
// 	return p.Impl.IdempotentInit(p.close)
// }
// where p.close() is contains the real close code for your plugin
func (i *Impl) Close() error { return nil }

// IdempotentInit increments the refCount and calls init precisely once
func (i *Impl) IdempotentInit(init func() error) error {
	i.refCountMutex.Lock()
	defer i.refCountMutex.Unlock()
	if i.refCount < 0 { // i.refCount < 0 means we are terminally closed and no longer Init-able
		return errors.New(ReinitErrorStr)
	}
	i.refCount++
	if i.refCount == 1 {
		i.initErr = init()
	}
	return i.initErr
}

// IdempotentClose decrements the refCount and calls close precisely once
// when refCount is equal to zero.
func (i *Impl) IdempotentClose(close func() error) (err error) {
	i.refCountMutex.Lock()
	defer i.refCountMutex.Unlock()
	if i.refCount < 0 { // i.refCount < 0 means we are terminally closed and no longer Init-able
		return i.closeErr
	}
	i.refCount--
	if i.refCount == 0 {
		i.closeErr = close()
		i.refCount-- // Make sure refcount < 0
	}
	return i.closeErr
}
