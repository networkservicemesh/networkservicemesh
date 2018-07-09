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
	"testing"

	"github.com/ligato/networkservicemesh/plugins/idempotent"
	"github.com/ligato/networkservicemesh/utils/helper"
	id1 "github.com/ligato/networkservicemesh/utils/idempotent"
	. "github.com/onsi/gomega"
)

type API interface {
	Running() bool
}

type PluginAPI interface {
	idempotent.PluginAPI
	API
}

type Runable struct {
	running bool
}

type Plugin struct {
	id1.Impl
	Runable
}

func (p *Runable) Running() bool { return p.running }

func (p *Plugin) Init() error  { p.running = true; return nil }
func (p *Plugin) Close() error { p.running = false; return nil }

func TestCheckDepsNil(t *testing.T) {
	RegisterTestingT(t)
	Expect(helper.CheckDeps(nil)).ToNot(Succeed())
}

type PluginString string

func (*PluginString) Init() error        { return nil }
func (*PluginString) Close() error       { return nil }
func (*PluginString) IsIdempotent() bool { return true }
func (*PluginString) State() id1.State   { return id1.NEW }

func TestCheckDepsNonStruct(t *testing.T) {
	RegisterTestingT(t)
	p := PluginString("Foo")
	Expect(helper.CheckDeps(&p)).ToNot(Succeed())
}

func TestCheckDepsNoDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &Plugin{}
	Expect(helper.CheckDeps(p)).To(Succeed())
}

type PluginStringDeps struct {
	id1.Impl
	Deps string
}

func TestCheckDepsNonStructDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginStringDeps{Deps: "foo"}
	Expect(helper.CheckDeps(p)).ToNot(Succeed())
}

type PluginNonInterfaceDeps struct {
	id1.Impl
	Deps struct{ Foo string }
}

type PrivateDeps struct {
	one API
}

type PluginPrivateDep struct {
	id1.Impl
	Deps PrivateDeps
}

func TestCheckDepsPrivatedeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginPrivateDep{}
	Expect(helper.CheckDeps(p)).ToNot(Succeed())
}

type NonOptionalDeps struct {
	One API
}
type PluginWithDeps struct {
	id1.Impl
	Deps NonOptionalDeps
}

func TestCheckDepsUnsetDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{}
	Expect(helper.CheckDeps(p)).ToNot(Succeed())
}

type OptionalDeps struct {
	One API `optional:"true"`
}

type PluginWithOptionalDep struct {
	id1.Impl
	Deps OptionalDeps
}

func TestCheckDepsUnsetOptionalDep(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithOptionalDep{}
	Expect(helper.CheckDeps(p)).To(Succeed())
}

func TestCheckDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{
		Deps: NonOptionalDeps{
			One: &Plugin{},
		},
	}
	Expect(helper.CheckDeps(p)).To(Succeed())
}
