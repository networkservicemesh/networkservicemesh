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

package helper_test

import (
	"fmt"
	"testing"

	"github.com/ligato/networkservicemesh/utils/helper"

	. "github.com/onsi/gomega"
)

func TestCloseDeps(t *testing.T) {
	RegisterTestingT(t)
	one := &Plugin{}
	p := &PluginWithDeps{
		Deps: NonOptionalDeps{
			One: one,
		},
	}

	Expect(one.Init()).To(Succeed())
	Expect(p.Deps.One.Running()).To(BeTrue())
	Expect(helper.CloseDeps(p)).To(Succeed())
	Expect(p.Deps.One.Running()).To(BeFalse())
}

func TestCloseDepsUnSetDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{}
	Expect(helper.CloseDeps(p)).ToNot(Succeed())
}

func TestCloseNoDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &Plugin{}
	Expect(helper.CloseDeps(p)).To(Succeed())
}

func TestCloseDepsNonPluginDep(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{
		Deps: NonOptionalDeps{
			One: &Runable{},
		},
	}
	Expect(helper.CloseDeps(p)).To(Succeed())
	Expect(p.Deps.One.Running()).To(BeFalse())
}

type PluginErrorOnClose struct {
	Plugin
}

func (*PluginErrorOnInit) Close() error { return fmt.Errorf("PluginErrorOnInit always fails on Init") }

func TestCloseDepsErrorOnInit(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{
		Deps: NonOptionalDeps{
			One: &PluginErrorOnInit{},
		},
	}
	Expect(helper.CloseDeps(p)).ToNot(Succeed())
	Expect(p.Deps.One.Running()).To(BeFalse())
}
