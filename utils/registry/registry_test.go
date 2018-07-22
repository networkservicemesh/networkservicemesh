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

package registry_test

import (
	"testing"

	. "github.com/onsi/gomega"

	id1 "github.com/ligato/networkservicemesh/utils/idempotent"
	"github.com/ligato/networkservicemesh/utils/registry"
	"github.com/ligato/networkservicemesh/utils/registry/testsuites"
)

type Deps struct {
	Bool bool
}

type Plugin struct {
	id1.Impl
	Deps
}

func (p *Plugin) Close() error { registry.Shared().Delete(p); return nil }

type PluginNoDeps struct {
	id1.Impl
}

func (p *PluginNoDeps) Close() error { registry.Shared().Delete(p); return nil }

func TestRegistryReuse(t *testing.T) {
	RegisterTestingT(t)
	p1 := &Plugin{}
	p2 := &Plugin{}
	testsuites.SuiteRegistryReuse(t, p1, p2)
}

func TestRegistryDeleteOnClose(t *testing.T) {
	RegisterTestingT(t)
	p1 := &Plugin{}
	p2 := &Plugin{}
	testsuites.SuiteRegistryDeleteOnClose(t, p1, p2)
}

func TestRegistryUnequalPlugins(t *testing.T) {
	RegisterTestingT(t)
	p1 := &Plugin{}
	p2 := &Plugin{
		Deps: Deps{
			Bool: true,
		},
	}
	testsuites.SuiteRegistryUnequalPlugins(t, p1, p2)
}

func TestRegistry(t *testing.T) {
	RegisterTestingT(t)
	p1 := &Plugin{}
	p2 := &Plugin{}
	p3 := &Plugin{
		Deps: Deps{
			Bool: true,
		},
	}
	testsuites.SuiteRegistry(t, p1, p2, p3)
}

func TestRegistryNoDeps(t *testing.T) {
	RegisterTestingT(t)
	p1 := &Plugin{}
	p2 := &Plugin{}
	testsuites.SuiteRegistryNoDeps(t, p1, p2)
}
