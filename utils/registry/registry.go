// Copyright 2018 Red Hat, Inc.
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

package registry

import (
	"reflect"
	"sync"

	"github.com/ligato/networkservicemesh/plugins/idempotent"
	"github.com/ligato/networkservicemesh/utils/helper/deptools"
)

// Impl - implementation of a Registry
type Impl struct {
	plugins []idempotent.PluginAPI
	sync.Mutex
}

// LoadOrStore - return a plugin with Deps matching p.Deps if in registry
// or store p in the registry and return p if not
func (r *Impl) LoadOrStore(p idempotent.PluginAPI) idempotent.PluginAPI {
	r.Lock()
	defer r.Unlock()
	for _, value := range r.plugins {
		vdeps, _ := deptools.Get(value)
		pdeps, _ := deptools.Get(p)
		if reflect.TypeOf(value) == reflect.TypeOf(p) && reflect.DeepEqual(pdeps, vdeps) {
			return value
		}
	}
	r.plugins = append(r.plugins, p)
	return p
}

// Delete - delete p from registry
func (r *Impl) Delete(p idempotent.PluginAPI) {
	r.Lock()
	defer r.Unlock()
	for i, value := range r.plugins {
		vdeps, _ := deptools.Get(value)
		pdeps, _ := deptools.Get(p)
		if reflect.TypeOf(value) == reflect.TypeOf(p) && reflect.DeepEqual(pdeps, vdeps) {
			r.plugins = append(r.plugins[:i], r.plugins[i+1:]...)
			return
		}
	}
}

// Size - number of plugins in registry
func (r *Impl) Size() int {
	return len(r.plugins)
}

var registry *Impl
var once sync.Once

// Shared - Return the Shared registry
func Shared() *Impl {
	once.Do(func() { registry = New() })
	return registry
}

// New - return a new registry
func New() *Impl {
	return &Impl{}
}

// Testable - Wrapper around Impl - only for use in testing
type Testable struct {
	*Impl
}

// Clear is intended only for testing - only for use in testing
func (r *Testable) Clear() {
	r.plugins = nil
}

// WrapTestable - Wrap Impl in Testable - only for use in testing
func WrapTestable(i *Impl) *Testable {
	return &Testable{Impl: i}
}
