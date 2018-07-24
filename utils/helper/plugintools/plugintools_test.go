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

package plugintools_test

import (
	"fmt"
	"testing"

	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/utils/helper/plugintools"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	. "github.com/onsi/gomega"
)

type Deps struct {
	Name string
	Log  logger.FieldLogger `empty_value_ok:"true"`
}

type Plugin struct {
	idempotent.Impl
	Deps
}

func TestInitCheckFailMissingDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &Plugin{}
	Expect(plugintools.Init(p)).ToNot(Succeed())
}

func TestInitCheckPassCompeteDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &Plugin{
		Deps: Deps{
			Name: "foo",
		},
	}
	Expect(plugintools.Init(p)).To(Succeed())
}

func returnError() error { return fmt.Errorf("error(): error: init can never succeed") }

func TestInitCloseError(t *testing.T) {
	RegisterTestingT(t)
	p := &Plugin{
		Deps: Deps{
			Name: "foo",
		},
	}
	Expect(plugintools.Init(p, returnError)).ToNot(Succeed())
	Expect(plugintools.Close(p, returnError)).ToNot(Succeed())
}

func returnNil() error { return nil }

func TestInitCloseNoError(t *testing.T) {
	RegisterTestingT(t)
	p := &Plugin{
		Deps: Deps{
			Name: "foo",
		},
	}
	Expect(plugintools.Init(p, returnNil)).To(Succeed())
	Expect(plugintools.Close(p, returnNil)).To(Succeed())
}

type PluginUsingInitCloseFunc struct {
	idempotent.Impl
	Deps
	Running bool
}

func (p *PluginUsingInitCloseFunc) Init() error {
	return p.IdempotentInit(plugintools.InitFunc(p, p.init))
}

func (p *PluginUsingInitCloseFunc) init() error {
	p.Running = true
	return nil
}

func (p *PluginUsingInitCloseFunc) Close() error {
	return p.IdempotentClose(plugintools.CloseFunc(p, p.close))
}

func (p *PluginUsingInitCloseFunc) close() error {
	p.Running = false
	return nil
}

func TestInitCloseFunc(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginUsingInitCloseFunc{
		Deps: Deps{
			Name: "bar",
		},
	}
	Expect(p.Running).To(BeFalse())
	Expect(p.Init()).To(Succeed())
	Expect(p.Running).To(BeTrue())
	Expect(p.Close()).To(Succeed())
	Expect(p.Running).To(BeFalse())
}

type PluginUsingInitCloseLoggingFunc struct {
	PluginUsingInitCloseFunc
}

func (p *PluginUsingInitCloseLoggingFunc) Init() error {
	return p.IdempotentInit(plugintools.LoggingInitFunc(p.Deps.Log, p, p.init))
}

func (p *PluginUsingInitCloseLoggingFunc) Close() error {
	return p.IdempotentClose(plugintools.LoggingCloseFunc(p.Log, p, p.close))
}

func TestLoggingInitCloseFuncNilLogger(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginUsingInitCloseLoggingFunc{
		PluginUsingInitCloseFunc{
			Deps: Deps{
				Name: "bar",
			},
		},
	}
	Expect(p.Running).To(BeFalse())
	Expect(p.Init()).ToNot(Succeed())
	Expect(p.Running).To(BeFalse())
	Expect(p.Close()).ToNot(Succeed())
	Expect(p.Running).To(BeFalse())
}

func TestLoggingInitCloseFunc(t *testing.T) {
	RegisterTestingT(t)

	p := &PluginUsingInitCloseLoggingFunc{
		PluginUsingInitCloseFunc{
			Deps: Deps{
				Name: "bar",
				Log:  logger.ByName("bar"),
			},
		},
	}

	Expect(p.Running).To(BeFalse())
	Expect(p.Init()).To(Succeed())
	Expect(p.Running).To(BeTrue())
	Expect(p.Close()).To(Succeed())
	Expect(p.Running).To(BeFalse())
}

type PluginUsingInitCloseLoggingFuncFailed struct {
	PluginUsingInitCloseFunc
}

func (p *PluginUsingInitCloseLoggingFuncFailed) Init() error {
	return p.IdempotentInit(plugintools.LoggingInitFunc(p.Deps.Log, p, returnError, p.init))
}

func (p *PluginUsingInitCloseLoggingFuncFailed) Close() error {
	return p.IdempotentClose(plugintools.LoggingCloseFunc(p.Log, p, returnError, p.close))
}
func TestLoggingInitCloseFuncFailed(t *testing.T) {
	RegisterTestingT(t)

	p := &PluginUsingInitCloseLoggingFuncFailed{
		PluginUsingInitCloseFunc{
			Deps: Deps{
				Name: "bar",
				Log:  logger.ByName("bar"),
			},
		},
	}

	Expect(p.Running).To(BeFalse())
	Expect(p.Init()).ToNot(Succeed())
	Expect(p.Running).To(BeFalse())
	Expect(p.Close()).ToNot(Succeed())
	Expect(p.Running).To(BeFalse())
}

type FailedLogger struct {
	logger.FieldLoggerPlugin
}

func (p *FailedLogger) Init() error {
	return plugintools.LoggingInitFunc(p, p, p.FieldLoggerPlugin.Init, returnError)()
}

func (p *FailedLogger) Close() error {
	return plugintools.LoggingCloseFunc(p, p, p.FieldLoggerPlugin.Close, returnError)()
}

func TestPluginWithFailedDeps(t *testing.T) {
	RegisterTestingT(t)
	p := &PluginUsingInitCloseLoggingFunc{
		PluginUsingInitCloseFunc{
			Deps: Deps{
				Name: "bar",
				Log: &FailedLogger{
					FieldLoggerPlugin: logger.ByName("bar"),
				},
			},
		},
	}
	Expect(p.Running).To(BeFalse())
	Expect(p.Init()).ToNot(Succeed())
	Expect(p.Running).To(BeFalse())
	Expect(p.Close()).ToNot(Succeed())
	Expect(p.Running).To(BeFalse())

}
