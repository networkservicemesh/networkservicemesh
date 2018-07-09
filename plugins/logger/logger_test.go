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

package logger_test

import (
	"bytes"
	"encoding/json"
	"github.com/ligato/networkservicemesh/plugins/logger"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
)

func TestLoggerDefaultDeps(t *testing.T) {
	RegisterTestingT(t)
	var buffer bytes.Buffer
	var fields logrus.Fields
	formatter := &logrus.JSONFormatter{}
	logger.SetDefaultFormatter(formatter)
	logger.SetDefaultOut(&buffer)
	plugin := logger.NewPlugin()
	Expect(plugin).ToNot(BeNil())
	Expect(plugin.Deps.Formatter).To(Equal(formatter))
	Expect(plugin.Deps.Out).To(Equal(&buffer))
	Expect(plugin.Name).To(Equal(logger.DefaultName))
	err := plugin.Init()
	Expect(err).To(BeNil())
	plugin.Info("Foo")
	err = json.Unmarshal(buffer.Bytes(), &fields)
	Expect(err).To(BeNil())
	Expect(fields[logger.LogNameFieldName]).To(Equal(logger.DefaultName))
	Expect(fields["msg"]).To(Equal("Foo"))
	Expect(fields["pid"]).ToNot(BeNil())
	Expect(fields["source"]).ToNot(BeNil())
	err = plugin.Close()
	Expect(err).To(BeNil())
}

func TestLoggerCustomDeps(t *testing.T) {
	RegisterTestingT(t)
	var buffer bytes.Buffer
	var fields logrus.Fields
	formatter := &logrus.JSONFormatter{}
	name := "customlogger"
	deps := &logger.Deps{Out: &buffer, Formatter: formatter, Name: name}
	plugin := logger.NewPlugin(logger.UseDeps(deps))
	Expect(plugin).ToNot(BeNil())
	Expect(plugin.Deps.Formatter).To(Equal(formatter))
	Expect(plugin.Deps.Out).To(Equal(&buffer))
	Expect(plugin.Name).To(Equal(name))
	err := plugin.Init()
	Expect(err).To(BeNil())
	plugin.Info("Foo")
	err = json.Unmarshal(buffer.Bytes(), &fields)
	Expect(err).To(BeNil())
	Expect(fields[logger.LogNameFieldName]).To(Equal(name))
	Expect(fields["msg"]).To(Equal("Foo"))
	Expect(fields["pid"]).ToNot(BeNil())
	Expect(fields["source"]).ToNot(BeNil())
}

func TestDefaultFormatterOut(t *testing.T) {
	RegisterTestingT(t)
	logger.SetDefaultFormatter(nil)
	logger.SetDefaultOut(nil)
	plugin := logger.NewPlugin()
	Expect(plugin).ToNot(BeNil())
	Expect(plugin.Deps.Formatter).To(Equal(new(logrus.TextFormatter)))
	Expect(plugin.Deps.Out).To(Equal(os.Stderr))
	Expect(plugin.Name).To(Equal(logger.DefaultName))
}

func TestFields(t *testing.T) {
	RegisterTestingT(t)
	var buffer bytes.Buffer
	var fields logrus.Fields
	formatter := &logrus.JSONFormatter{}
	logger.SetDefaultFormatter(formatter)
	logger.SetDefaultOut(&buffer)
	f := logrus.Fields{"animal": "walrus"}
	plugin := logger.NewPlugin(logger.UseDeps(&logger.Deps{Fields: f}))
	Expect(plugin).ToNot(BeNil())
	Expect(plugin.Deps.Formatter).To(Equal(formatter))
	Expect(plugin.Deps.Out).To(Equal(&buffer))
	Expect(plugin.Name).To(Equal(logger.DefaultName))
	err := plugin.Init()
	Expect(err).To(BeNil())
	plugin.Info("Foo")
	err = json.Unmarshal(buffer.Bytes(), &fields)
	Expect(err).To(BeNil())
	Expect(fields[logger.LogNameFieldName]).To(Equal(logger.DefaultName))
	Expect(fields["animal"]).To(Equal("walrus"))
	Expect(fields["msg"]).To(Equal("Foo"))
	Expect(fields["pid"]).ToNot(BeNil())
	Expect(fields["source"]).ToNot(BeNil())
	err = plugin.Close()
	Expect(err).To(BeNil())
}

func TestUseLevel(t *testing.T) {
	RegisterTestingT(t)
	var buffer bytes.Buffer
	var fields logrus.Fields
	formatter := &logrus.JSONFormatter{}
	logger.SetDefaultFormatter(formatter)
	logger.SetDefaultOut(&buffer)

	plugin := logger.NewPlugin(logger.UseLevel(logrus.DebugLevel))
	Expect(plugin).ToNot(BeNil())
	Expect(plugin.Deps.Formatter).To(Equal(formatter))
	Expect(plugin.Deps.Out).To(Equal(&buffer))
	Expect(plugin.Name).To(Equal(logger.DefaultName))
	err := plugin.Init()
	Expect(err).To(BeNil())
	plugin.Debug("Foo")
	err = json.Unmarshal(buffer.Bytes(), &fields)
	Expect(err).To(BeNil())
	Expect(fields[logger.LogNameFieldName]).To(Equal(logger.DefaultName))
	Expect(fields["msg"]).To(Equal("Foo"))
	Expect(fields["pid"]).ToNot(BeNil())
	Expect(fields["source"]).ToNot(BeNil())
	err = plugin.Close()
	Expect(err).To(BeNil())
}

func TestUseConfig(t *testing.T) {
	RegisterTestingT(t)
	var buffer bytes.Buffer
	var fields logrus.Fields
	formatter := &logrus.JSONFormatter{}
	logger.SetDefaultFormatter(formatter)
	logger.SetDefaultOut(&buffer)
	f := logrus.Fields{"animal": "walrus"}
	selector := map[string]string{logger.LogNameFieldName: logger.DefaultName, "animal": "walrus"}
	ce1 := logger.ConfigEntry{Level: logrus.DebugLevel, Selector: selector}
	selector = map[string]string{logger.LogNameFieldName: logger.DefaultName, "animal": "walrus"}
	ce2 := logger.ConfigEntry{Level: logrus.PanicLevel, Selector: selector}
	selector = map[string]string{"animal": "walrus"}
	ce3 := logger.ConfigEntry{Level: logrus.FatalLevel, Selector: selector}
	selector = map[string]string{"animal": "cat"}
	ce4 := logger.ConfigEntry{Level: logrus.FatalLevel, Selector: selector}
	ces := []logger.ConfigEntry{ce1, ce2, ce3, ce4}
	config := &logger.Config{ConfigEntries: ces}
	plugin := logger.NewPlugin(logger.UseDeps(&logger.Deps{Fields: f}), logger.UseConfig(config))
	Expect(plugin).ToNot(BeNil())
	Expect(plugin.Deps.Formatter).To(Equal(formatter))
	Expect(plugin.Deps.Out).To(Equal(&buffer))
	Expect(plugin.Name).To(Equal(logger.DefaultName))
	err := plugin.Init()
	Expect(err).To(BeNil())
	Expect(plugin.Log).NotTo(BeNil())
	Expect(plugin.Log.Level).To(Equal(logrus.DebugLevel))
	plugin.Debug("Foo")
	err = json.Unmarshal(buffer.Bytes(), &fields)
	Expect(err).To(BeNil())
	Expect(fields[logger.LogNameFieldName]).To(Equal(logger.DefaultName))
	Expect(fields["animal"]).To(Equal("walrus"))
	Expect(fields["msg"]).To(Equal("Foo"))
	Expect(fields["pid"]).ToNot(BeNil())
	Expect(fields["source"]).ToNot(BeNil())
	err = plugin.Close()
	Expect(err).To(BeNil())
}

func TestSharedPlugin(t *testing.T) {
	RegisterTestingT(t)
	plugin1 := logger.SharedPlugin()
	Expect(plugin1).NotTo(BeNil())
	plugin2 := logger.SharedPlugin()
	Expect(plugin2).To(Equal(plugin1))
}

type Hook struct{}

// Levels to which to add pid.Hook
func (hook *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire when logging and logrus.Entry to which this Hook applies
func (hook *Hook) Fire(entry *logrus.Entry) error {
	entry.Data["hook"] = "true"
	return nil
}

func TestHooks(t *testing.T) {
	RegisterTestingT(t)
	var buffer bytes.Buffer
	var fields logrus.Fields
	formatter := &logrus.JSONFormatter{}
	logger.SetDefaultFormatter(formatter)
	logger.SetDefaultOut(&buffer)
	plugin := logger.NewPlugin(logger.UseDeps(&logger.Deps{Hooks: []logrus.Hook{&Hook{}}}))
	Expect(plugin).ToNot(BeNil())
	Expect(plugin.Deps.Formatter).To(Equal(formatter))
	Expect(plugin.Deps.Out).To(Equal(&buffer))
	Expect(plugin.Name).To(Equal(logger.DefaultName))
	err := plugin.Init()
	Expect(err).To(BeNil())
	plugin.Info("Foo")
	err = json.Unmarshal(buffer.Bytes(), &fields)
	Expect(err).To(BeNil())
	Expect(fields[logger.LogNameFieldName]).To(Equal(logger.DefaultName))
	Expect(fields["msg"]).To(Equal("Foo"))
	Expect(fields["pid"]).ToNot(BeNil())
	Expect(fields["source"]).ToNot(BeNil())
	Expect(fields["hook"]).To(Equal("true"))
	err = plugin.Close()
	Expect(err).To(BeNil())
}

func TestNoMatchingSelector(t *testing.T) {
	RegisterTestingT(t)
	var buffer bytes.Buffer
	formatter := &logrus.JSONFormatter{}
	logger.SetDefaultFormatter(formatter)
	logger.SetDefaultOut(&buffer)
	f := logrus.Fields{"animal": "walrus"}
	selector := map[string]string{logger.LogNameFieldName: logger.DefaultName, "animal": "cat"}
	ce1 := logger.ConfigEntry{Level: logrus.DebugLevel, Selector: selector}
	ces := []logger.ConfigEntry{ce1}
	config := &logger.Config{ConfigEntries: ces}
	plugin := logger.NewPlugin(logger.UseDeps(&logger.Deps{Fields: f}), logger.UseConfig(config))
	Expect(plugin).ToNot(BeNil())
	Expect(plugin.Deps.Formatter).To(Equal(formatter))
	Expect(plugin.Deps.Out).To(Equal(&buffer))
	Expect(plugin.Name).To(Equal(logger.DefaultName))
	err := plugin.Init()
	Expect(err).To(BeNil())
	Expect(plugin.Log).NotTo(BeNil())
	Expect(plugin.Log.Level).To(Equal(logrus.InfoLevel))

}
