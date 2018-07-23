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
	"testing"

	"github.com/ligato/networkservicemesh/utils/helper/deptools"
	id1 "github.com/ligato/networkservicemesh/utils/idempotent"
	. "github.com/onsi/gomega"
)

type API interface {
	Running() bool
}

type Runable struct {
	running bool
}

type Plugin struct {
	id1.Impl
	Runable
}

func (p *Runable) Running() bool { return p.running }

func (p *Plugin) Init() error {
	return p.IdempotentInit(func() error { p.running = true; return nil })
}
func (p *Plugin) Close() error {
	return p.IdempotentClose(func() error { p.running = false; return nil })
}

func TestCheckNil(t *testing.T) {
	RegisterTestingT(t)
	Expect(deptools.Check(nil)).ToNot(Succeed())
}

type PluginString string

func (*PluginString) Init() error        { return nil }
func (*PluginString) Close() error       { return nil }
func (*PluginString) IsIdempotent() bool { return true }
func (*PluginString) State() id1.State   { return id1.NEW }

func TestCheckNonStruct(t *testing.T) {
	RegisterTestingT(t)
	p := PluginString("Foo")
	Expect(deptools.Check(&p)).ToNot(Succeed())
}

func TestCheckNoDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &Plugin{}
	Expect(deptools.Check(p)).To(Succeed())
}

type PluginStringDeps struct {
	id1.Impl
	Deps string
}

func TestCheckNonStructDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginStringDeps{Deps: "foo"}
	Expect(deptools.Check(p)).ToNot(Succeed())
}

type PrivateDeps struct {
	one API
}

type PluginPrivateDep struct {
	id1.Impl
	Deps PrivateDeps
}

func TestCheckPrivatedeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginPrivateDep{}
	Expect(deptools.Check(p)).ToNot(Succeed())
}

type NonOptionalDeps struct {
	One API
}
type PluginWithDeps struct {
	id1.Impl
	Deps NonOptionalDeps
}

func TestCheckUnsetDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{}
	Expect(deptools.Check(p)).ToNot(Succeed())
}

type OptionalDeps struct {
	One API `optional:"true"`
}

type PluginWithOptionalDep struct {
	id1.Impl
	Deps OptionalDeps
}

func TestCheckUnsetOptionalDep(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithOptionalDep{}
	Expect(deptools.Check(p)).To(Succeed())
}

func TestCheck(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginWithDeps{
		Deps: NonOptionalDeps{
			One: &Plugin{},
		},
	}
	Expect(deptools.Check(p)).To(Succeed())
}

type DepsWithName struct {
	Name string
}

type PluginDepsWithName struct {
	id1.Impl
	Deps DepsWithName
}

func TestCheckDepsWithName(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginDepsWithName{}
	Expect(deptools.Check(p)).ToNot(Succeed())
}
