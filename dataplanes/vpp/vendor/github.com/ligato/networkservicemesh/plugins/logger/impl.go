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

package logger

import (
	"io"
	"os"

	"github.com/go-errors/errors"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	"github.com/ligato/networkservicemesh/utils/registry"
	"github.com/sirupsen/logrus"
)

// Plugin -  Providies a logger
//         Deps: plugin dependencies
//         *Config: Pointer to logger.Config
type Plugin struct {
	idempotent.Impl
	Deps
	*Config
	logrus.FieldLogger
	Log *logrus.Logger
}

// Init Log.Plugin
func (p *Plugin) Init() error {
	return p.Impl.IdempotentInit(p.init)
}

func (p *Plugin) init() error {
	err := p.Deps.ConfigLoader.Init()
	if err != nil {
		return err
	}
	if p.Config == nil {
		p.Config = p.Deps.ConfigLoader.LoadConfig().(*Config)
	}

	for _, hook := range p.Hooks {
		p.Log.AddHook(hook)
	}
	p.FieldLogger = p.Log.WithFields(p.Fields)
	p.setLogLevel()
	p.Log.Formatter = p.Deps.Formatter
	p.Log.Out = p.Deps.Out

	return nil
}

func (p *Plugin) setLogLevel() {
	var matchEntry *ConfigEntry // Current matching config entry
	matchDegree := -1           // How many labels match
	for _, entry := range p.Config.ConfigEntries {
		match := true // Presume we match
		for key, value := range entry.Selector {
			fieldValue, ok := p.Fields[key]
			if !ok || value != fieldValue {
				match = false // We don't match
				break
			}
		}
		// If we match more than we previously matched, go with that
		degree := len(entry.Selector)
		if match && degree > matchDegree {
			// We must copy the match entry because entry simply gets overwritten every pass through the loop
			e := entry
			matchEntry = &e
			matchDegree = len(entry.Selector)
		}
	}
	// If we have a match and a Log, set the Log Level from it
	if matchEntry != nil && p.Log != nil {
		p.Log.SetLevel(matchEntry.Level)
		return
	}
	p.Log.SetLevel(logrus.InfoLevel)
}

// Close Log.Plugin
func (p *Plugin) Close() error {
	return p.Impl.IdempotentClose(p.close)
}

func (p *Plugin) close() error {
	registry.Shared().Delete(p)
	return p.Deps.ConfigLoader.Close()
}

func (p *Plugin) WithStackTrace(error *errors.Error) logrus.FieldLogger {
	return p.FieldLogger.WithField("callstack", error.ErrorStack()).WithError(error)
}

var defaultFormatter logrus.Formatter

func DefaultFormatter() logrus.Formatter {
	if defaultFormatter == nil {
		defaultFormatter = new(logrus.TextFormatter)
	}
	return defaultFormatter
}

func SetDefaultFormatter(f logrus.Formatter) {
	defaultFormatter = f
}

var defaultOut io.Writer

func DefaultOut() io.Writer {
	if defaultOut == nil {
		defaultOut = os.Stderr
	}
	return defaultOut
}

func SetDefaultOut(o io.Writer) {
	defaultOut = o
}
