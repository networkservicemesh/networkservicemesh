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

package dataplaneregistrar

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/ligato/networkservicemesh/plugins/nsmserver"
	"github.com/ligato/networkservicemesh/utils/registry/testsuites"
)

func TestSharedPlugin(t *testing.T) {
	RegisterTestingT(t)
	plugin1 := nsmserver.SharedPlugin()
	Expect(plugin1).NotTo(BeNil())
	plugin2 := nsmserver.SharedPlugin()
	Expect(plugin2 == plugin1).To(BeTrue())
}

func TestNewPluginNotShared(t *testing.T) {
	RegisterTestingT(t)
	plugin1 := nsmserver.NewPlugin()
	Expect(plugin1).NotTo(BeNil())
	plugin2 := nsmserver.SharedPlugin()
	Expect(plugin2 == plugin1).ToNot(BeTrue())
}

// This really should be tested, but right now plugin.Close() is timing out

func TestSharedPluginRemoveOnClose(t *testing.T) {
	RegisterTestingT(t)
	plugin1 := nsmserver.SharedPlugin()
	plugin1.Init()
	plugin1.Close()
	plugin2 := nsmserver.SharedPlugin()
	Expect(plugin2 == plugin1).ToNot(BeTrue())
}

func TestNonDefaultName(t *testing.T) {
	RegisterTestingT(t)
	name := "foo"
	plugin := nsmserver.NewPlugin(nsmserver.UseDeps(&nsmserver.Deps{Name: name}))
	Expect(plugin).NotTo(BeNil())
}

func TestWithRegistry(t *testing.T) {
	name := "foo"
	p1 := nsmserver.NewPlugin()
	p2 := nsmserver.NewPlugin()
	p3 := nsmserver.NewPlugin(nsmserver.UseDeps(&nsmserver.Deps{Name: name}))
	testsuites.SuiteRegistry(t, p1, p2, p3)
}
