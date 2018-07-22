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

package testsuites

import (
	"reflect"
	"testing"

	"github.com/ligato/networkservicemesh/plugins/idempotent"
	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	"github.com/ligato/networkservicemesh/utils/registry"
	. "github.com/onsi/gomega"
)

// SuiteRegistry - Test all aspects of your plugin with the registry
// t = *testing.T
// p1, p2 - Two Plugins that are *not* identical, but should have the same Deps values
// p3 - A Plugin with different Dep values than p1 or p2
func SuiteRegistry(t *testing.T, p1 idempotent.PluginAPI, p2 idempotent.PluginAPI, p3 idempotent.PluginAPI) {
	RegisterTestingT(t)
	SuiteRegistryReuse(t, p1, p2)
	SuiteRegistryUnequalPlugins(t, p1, p3)
	SuiteRegistryDeleteOnClose(t, p1, p2)
}

// SuiteRegistryNoDeps - A Registry Suite to use for Plugins with NoDeps
// t = *testing.T
// p1, p2 - Two Plugins that are *not* identical, but should have the same Deps values
func SuiteRegistryNoDeps(t *testing.T, p1 idempotent.PluginAPI, p2 idempotent.PluginAPI) {
	RegisterTestingT(t)
	SuiteRegistryReuse(t, p1, p2)
	SuiteRegistryDeleteOnClose(t, p1, p2)
}

// SuiteRegistryReuse :
// t = *testing.T
// p1, p2 - Two Plugins that are *not* identical, but should have the same Deps values
func SuiteRegistryReuse(t *testing.T, p1 idempotent.PluginAPI, p2 idempotent.PluginAPI) {
	RegisterTestingT(t)
	r := registry.New()
	Expect(r.Size()).To(Equal(0), "Expected Empty Registry")
	Expect(p1 == p2).To(BeFalse(), "p1 == p2, p1 and p2 cannot be identical for this test")
	Expect(reflect.TypeOf(p1) == reflect.TypeOf(p2)).To(BeTrue(), "p1 and p2 are not of the same type")
	deps1, err := deptools.Get(p1)
	Expect(err).To(Succeed())
	deps2, err := deptools.Get(p2)
	Expect(err).To(Succeed())
	Expect(reflect.DeepEqual(deps1, deps2)).To(BeTrue(), "p1.Deps and p2.Deps are not deeply equal")
	r.LoadOrStore(p1)
	Expect(r.Size()).To(Equal(1), "Expected Registry to have 1 elements")
	p3 := r.LoadOrStore(p2)
	Expect(r.Size()).To(Equal(1), "Expected Registry to have 1 elements")
	Expect(p1 == p3).To(BeTrue())
	Expect(p2 == p3).To(BeFalse())
}

// SuiteRegistryDeleteOnClose :
// t = *testing.T
// p1, p2 - Two Plugins that are *not* identical, but should have the same Deps values
func SuiteRegistryDeleteOnClose(t *testing.T, p1 idempotent.PluginAPI, p2 idempotent.PluginAPI) {
	RegisterTestingT(t)
	r := registry.Shared()
	registry.WrapTestable(r).Clear()
	Expect(r.Size()).To(Equal(0), "Expected Empty Registry")
	Expect(p1 == p2).To(BeFalse())
	Expect(reflect.TypeOf(p1) == reflect.TypeOf(p2)).To(BeTrue())
	deps1, err := deptools.Get(p1)
	Expect(err).To(Succeed())
	deps2, err := deptools.Get(p2)
	Expect(err).To(Succeed())
	Expect(reflect.DeepEqual(deps1, deps2)).To(BeTrue(), "p1.Deps and p2.Deps are not deeply equal")
	r.LoadOrStore(p1)
	Expect(r.Size()).To(Equal(1), "Expected Registry to have 1 elements")
	// Note: Purposely not checking err from Init() or Close()
	// We just care here about proper operation of the Plugin with the Registry
	p1.Init()
	p1.Close()
	Expect(r.Size()).To(Equal(0), "Expected Empty Registry - you probably forget to call registry.Shared().Delete(p) on your plugin in it's Plugin.close() function")
	p3 := r.LoadOrStore(p2)
	Expect(r.Size()).To(Equal(1))
	Expect(p1 == p3).To(BeFalse())
	Expect(p2 == p3).To(BeTrue())
}

// SuiteRegistryUnequalPlugins :
// t = *testing.T
// p1, p2 - Two Plugins that have different Deps value
func SuiteRegistryUnequalPlugins(t *testing.T, p1 idempotent.PluginAPI, p2 idempotent.PluginAPI) {
	RegisterTestingT(t)
	registry := registry.New()
	Expect(registry.Size()).To(Equal(0), "Expected Empty Registry")
	Expect(p1 == p2).To(BeFalse())
	Expect(reflect.TypeOf(p1) == reflect.TypeOf(p2)).To(BeTrue())
	deps1, err := deptools.Get(p1)
	Expect(err).To(Succeed())
	deps2, err := deptools.Get(p2)
	Expect(err).To(Succeed())
	Expect(reflect.DeepEqual(deps1, deps2)).To(BeFalse(), "p1.Deps and p2.Deps must not be deeply equal for this test")
	registry.LoadOrStore(p1)
	Expect(registry.Size()).To(Equal(1), "Expected Registry to have 1 element")
	p3 := registry.LoadOrStore(p2)
	Expect(registry.Size()).To(Equal(2), "Expected Registry to have 1 element")
	Expect(p1 == p3).To(BeFalse())
	Expect(p2 == p3).To(BeTrue())
}
