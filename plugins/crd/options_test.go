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

package crd_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestSharedPlugin(t *testing.T) {
	RegisterTestingT(t)
	plugin1 := netmeshplugincrd.SharedPlugin()
	Expect(plugin1).NotTo(BeNil())
	plugin2 := netmeshplugincrd.SharedPlugin()
	Expect(plugin2 == plugin1).To(BeTrue())
}

func TestNewPluginNotShared(t *testing.T) {
	RegisterTestingT(t)
	plugin1 := netmeshplugincrd.NewPlugin()
	Expect(plugin1).NotTo(BeNil())
	plugin2 := netmeshplugincrd.SharedPlugin()
	Expect(plugin2 == plugin1).ToNot(BeTrue())
}

func TestSharedPluginRemoveOnClose(t *testing.T) {
	RegisterTestingT(t)
	plugin1 := netmeshplugincrd.SharedPlugin()
	plugin1.Init()
	plugin1.Close()
	plugin2 := netmeshplugincrd.SharedPlugin()
	Expect(plugin2 == plugin1).ToNot(BeTrue())
}

func TestNonDefaultName(t *testing.T) {
	RegisterTestingT(t)
	name := "foo"
	plugin := netmeshplugincrd.NewPlugin(netmeshplugincrd.UseDeps(&netmeshplugincrd.Deps{Name: name}))
	Expect(plugin).NotTo(BeNil())
}
