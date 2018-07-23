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

package deptools_test

import (
	"fmt"
	"testing"

	id1 "github.com/ligato/networkservicemesh/plugins/idempotent"
	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	. "github.com/onsi/gomega"
)

func TestInit(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{
		Deps: NonOptionalDeps{
			One: &Plugin{},
		},
	}
	Expect(deptools.Init(p)).To(Succeed())
	Expect(p.Deps.One.Running()).To(BeTrue())
}

func TestDepsUnSetDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{}
	Expect(deptools.Init(p)).ToNot(Succeed())
}

func TestInitNoDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &Plugin{}
	Expect(deptools.Init(p)).To(Succeed())
}

func TestInitNonPluginDep(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{
		Deps: NonOptionalDeps{
			One: &Runable{},
		},
	}
	Expect(deptools.Init(p)).To(Succeed())
	Expect(p.Deps.One.Running()).To(BeFalse())
}

type PluginErrorOnInit struct {
	Plugin
}

type TwoDeps struct {
	One id1.PluginAPI
	Two id1.PluginAPI
}

type PluginTwoDeps struct {
	idempotent.Impl
	Deps TwoDeps
}

func (*PluginErrorOnInit) Init() error { return fmt.Errorf("PluginErrorOnInit always fails on Init") }

func TestInitErrorOnInit(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginTwoDeps{
		Deps: TwoDeps{
			One: &Plugin{},
			Two: &PluginErrorOnInit{},
		},
	}
	Expect(deptools.Init(p)).ToNot(Succeed())
	Expect(p.Deps.One.State()).To(Equal(idempotent.CLOSED))
}
