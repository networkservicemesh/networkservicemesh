// Copyright (c) 2017 Cisco and/or its affiliates.
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

package core

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/namsral/flag"

	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/logrus"
)

var (
	// DefaultMaxStartupTime defines maximal duration of start for agent.
	DefaultMaxStartupTime = 15 * time.Second
)

// Agent implements startup & shutdown procedures.
type Agent struct {
	// plugin list
	plugins []*NamedPlugin
	logging.Logger
	// The field is set before initialization of every plugin with its name.
	currentlyProcessing string
	// agent's stopwatch
	timer Timer
}

// Timer holds all startup times.
type Timer struct {
	// The startup/initialization must take no longer that maxStartup.
	MaxStartupTime time.Duration
	// timers
	agentStart     time.Time
	initStart      time.Time
	afterInitStart time.Time
	// durations
	init      time.Duration
	afterInit time.Duration
}

// NewAgent returns a new instance of the Agent with plugins. Use options if needed:
// <WithLogger() option> will be used to log messages related to the agent life-cycle,
// but not for the plugins themselves.
// <WithTimeout() option> puts a time limit on initialization of all provided plugins.
// Agent.Start() returns ErrPluginsInitTimeout error if one or more plugins fail
// to initialize inside the specified time limit.
// <WithPlugins() option> is a variable list of plugins to load. ListPluginsInFlavor() helper
// method can be used to obtain the list from a given flavor.
//
// Example 1 (existing flavor - or use alias rpc.NewAgent()):
//
//    core.NewAgent(&FlavorRPC{}, core.WithTimeout(5 * time.Second), rpc.WithPlugins(func(flavor *FlavorRPC) []*core.NamedPlugins {
//		return []*core.NamedPlugins{{"customization": &CustomPlugin{DependencyXY: &flavor.GRPC}}}
//    })
//
// Example 2 (custom flavor):
//
//    core.NewAgent(&MyFlavor{}, core.WithTimeout(5 * time.Second), my.WithPlugins(func(flavor *MyFlavor) []*core.NamedPlugins {
//		return []*core.NamedPlugins{{"customization": &CustomPlugin{DependencyXY: &flavor.XY}}}
//    })
func NewAgent(flavor Flavor, opts ...Option) *Agent {
	maxStartup := DefaultMaxStartupTime

	var flavors []Flavor
	if fs, ok := flavor.(flavorAggregator); ok {
		flavors = fs.fs
	} else {
		flavors = []Flavor{flavor}
	}

	flavor.Inject()

	var agentCoreLogger logging.Logger
	plugins := flavor.Plugins()

	for _, opt := range opts {
		switch opt.(type) {
		case WithPluginsOpt:
			plugins = append(plugins, opt.(WithPluginsOpt).Plugins(flavors...)...)
		case *WithTimeoutOpt:
			ms := opt.(*WithTimeoutOpt).Timeout
			if ms > 0 {
				maxStartup = ms
			}
		case *WithLoggerOpt:
			agentCoreLogger = opt.(*WithLoggerOpt).Logger
		}
	}

	if logRegGet, ok := flavor.(logRegistryGetter); ok && logRegGet != nil {
		logReg := logRegGet.LogRegistry()
		if logReg != nil {
			agentCoreLogger = logReg.NewLogger("agentcore")
		} else {
			agentCoreLogger = logrus.DefaultLogger()
		}
	} else {
		agentCoreLogger = logrus.DefaultLogger()
	}

	return &Agent{
		plugins: plugins,
		Logger:  agentCoreLogger,
		timer: Timer{
			MaxStartupTime: maxStartup,
		},
	}
}

// NewAgentDeprecated older & deprecated version of a constructor
// Function returns a new instance of the Agent with plugins.
// <logger> will be used to log messages related to the agent life-cycle,
// but not for the plugins themselves.
// <maxStartup> sets a time limit for initialization of all provided plugins.
// Agent.Start() returns ErrPluginsInitTimeout error if one or more plugins fail
// to initialize in the specified time limit.
// <plugins> is a variable that holds a list of plugins to load. ListPluginsInFlavor() helper
// method can be used to obtain the list from a given flavor.
func NewAgentDeprecated(logger logging.Logger, maxStartup time.Duration, plugins ...*NamedPlugin) *Agent {
	return &Agent{
		plugins: plugins,
		Logger:  logger,
		timer: Timer{
			MaxStartupTime: maxStartup,
		},
	}
}

type logRegistryGetter interface {
	// LogRegistry is a getter for log registry instance
	LogRegistry() logging.Registry
}

// Start starts/initializes all selected plugins.
// The first iteration tries to run Init() method on every plugin from the list.
// If any of the plugins fails to initialize (Init() returns non-nil error),
// the initialization is cancelled by calling Close() method for already initialized
// plugins in the reverse order. The encountered error is returned by this
// function as-is.
// The second iteration does the same for the AfterInit() method. The difference
// is that AfterInit() is an optional method (not required by the Plugin
// interface, only suggested by PostInit interface) and therefore not necessarily
// called on every plugin.
// The startup/initialization must take no longer than maxStartup time limit,
// otherwise ErrPluginsInitTimeout error is returned.
func (agent *Agent) Start() error {
	agent.WithFields(logging.Fields{"CommitHash": CommitHash, "BuildDate": BuildDate}).
		Infof("Starting agent %v", BuildVersion)

	if !flag.Parsed() {
		flag.Parse()
	}

	doneChannel := make(chan struct{})
	errChannel := make(chan error)

	agent.timer.agentStart = time.Now()

	go func() {
		if err := agent.initPlugins(); err != nil {
			errChannel <- err
			return
		}

		if err := agent.handleAfterInit(); err != nil {
			errChannel <- err
			return
		}

		close(doneChannel)
	}()

	// Block until all Plugins are initialized or timeout expires.
	select {
	case <-doneChannel:
		agent.Infof("Agent started successfully, took %v (Init: %v, AfterInit: %v)",
			agent.timer.init+agent.timer.afterInit, agent.timer.init, agent.timer.afterInit)
		return nil

	case err := <-errChannel:
		agent.Debugf("Agent Init took %v", agent.timer.init)
		agent.Debugf("Agent AfterInit took %v", agent.timer.afterInit)
		return err

	case <-time.After(agent.timer.MaxStartupTime):
		if agent.timer.init == 0 {
			agent.Infof("Agent Init took > %v", agent.timer.MaxStartupTime)
		} else {
			agent.Infof("Agent Init took %v", agent.timer.init)
			agent.Infof("Agent AfterInit took > %v", agent.timer.MaxStartupTime)
		}
		return fmt.Errorf("plugin %s not completed before timeout", agent.currentlyProcessing)
	}
}

// Stop gracefully shuts down the Agent. It is called usually when the user
// interrupts the Agent from the EventLoopWithInterrupt().
//
// This implementation tries to call Close() method on every plugin on the list
// in the reverse order. It continues even if some error occurred.
func (agent *Agent) Stop() error {
	agent.Info("Stopping agent...")

	var errMsgs []string
	for i := len(agent.plugins) - 1; i >= 0; i-- {
		p := agent.plugins[i]

		agent.Debugf("Closing plugin: %s", p)

		if err := p.Plugin.Close(); err != nil {
			agent.Warnf("plugin %s: Close failed: %v", p, err)
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", p, err))
		}
	}

	agent.Info("Agent stopped")

	if len(errMsgs) > 0 {
		return errors.New(strings.Join(errMsgs, ", "))
	}

	return nil
}

// initPlugins calls Init() on all plugins in the list.
func (agent *Agent) initPlugins() error {
	// Flag indicates that some of the plugins failed to initialize
	var initPluginCounter int
	var wasError error

	agent.timer.initStart = time.Now()
	for index, plugin := range agent.plugins {
		initPluginCounter = index

		// Set currently initialized plugin name.
		agent.currentlyProcessing = plugin.String()

		// Skip all other plugins if some of them failed.
		if wasError != nil {
			agent.Warnf("plugin %s: Init skipped due to previous error", plugin)
			continue
		}

		pluginStartTime := time.Now()
		if err := plugin.Init(); err != nil {
			wasError = fmt.Errorf("plugin %s: Init failed: %v", plugin, err)
			agent.WithField("took", time.Since(pluginStartTime)).Error(wasError)
		} else {
			agent.WithField("took", time.Since(pluginStartTime)).Infof("plugin %s: Init ok", plugin)
		}
	}
	agent.timer.init = time.Since(agent.timer.initStart)

	if wasError != nil {
		//Stop the plugins that are initialized
		for i := initPluginCounter; i >= 0; i-- {
			p := agent.plugins[i]

			agent.Debugf("Closing plugin: %s", p)

			if err := p.Close(); err != nil {
				wasError = err
			}
		}
		return wasError
	}

	return nil
}

// handleAfterInit calls the AfterInit handlers on plugins that can only
// finish their initialization after all other plugins have been initialized.
func (agent *Agent) handleAfterInit() error {
	// Flag indicates that some of the plugins failed to after-initialize
	var wasError error

	agent.timer.afterInitStart = time.Now()
	for _, plugin := range agent.plugins {
		// Set currently after-initialized plugin name.
		agent.currentlyProcessing = plugin.String()

		// Skip all other plugins if some of them failed.
		if wasError != nil {
			agent.Warnf("plugin %s: AfterInit skipped due to previous error", plugin)
			continue
		}

		// Check if plugin implements AfterInit().
		if postPlugin, ok := plugin.Plugin.(PostInit); ok {
			pluginStartTime := time.Now()
			if err := postPlugin.AfterInit(); err != nil {
				wasError = fmt.Errorf("plugin %s: AfterInit failed: %v", plugin, err)
				agent.WithField("took", time.Since(pluginStartTime)).Error(wasError)
			} else {
				agent.WithField("took", time.Since(pluginStartTime)).Infof("plugin %s: AfterInit ok", plugin)
			}
		} else {
			agent.Debugf("plugin %s: no AfterInit implement", plugin)
		}
	}
	agent.timer.afterInit = time.Since(agent.timer.afterInitStart)

	if wasError != nil {
		agent.Stop()
		return wasError
	}

	return nil
}
