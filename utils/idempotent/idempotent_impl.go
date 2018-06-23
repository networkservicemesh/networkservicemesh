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
	"sync"
)

// IdemPotentImpl implements methods for wrapping Init() and Close() such that
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

// IdempotentInit increments the refCount and calls init precisely once
func (i *Impl) IdempotentInit(init func() error) error {
	i.refCountMutex.Lock()
	defer i.refCountMutex.Unlock()
	i.refCount++
	if i.refCount == 1 {
		i.initErr = init()
	}
	return i.initErr
}

// IdemPotentClose decrements the refCount and calls close precisely once
// when refCount is equal to zero.
func (i *Impl) IdempotentClose(close func() error) (err error) {
	i.refCountMutex.Lock()
	defer i.refCountMutex.Unlock()
	i.refCount--
	if i.refCount == 0 {
		i.closeErr = close()
	}
	return i.closeErr
}
