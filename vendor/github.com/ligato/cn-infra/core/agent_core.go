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
	"time"

	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/logrus"
	"github.com/ligato/cn-infra/utils/safeclose"
	"github.com/namsral/flag"
)

// variables set by the Makefile using ldflags
var (
	BuildVersion string
	BuildDate    string
	CommitHash   string
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

const (
	logErrorFmt        = "plugin %s: Init error '%s', took %v"
	logSuccessFmt      = "plugin %s: Init took %v"
	logSkippedFmt      = "plugin %s: Init skipped due to previous error"
	logAfterSkippedFmt = "plugin %s: AfterInit skipped due to previous error"
	logAfterErrorFmt   = "plugin %s: AfterInit error '%s', took %v"
	logAfterSuccessFmt = "plugin %s: AfterInit took %v"
	logNoAfterInitFmt  = "plugin %s: not implement AfterInit"
	logTimeoutFmt      = "plugin %s not completed before timeout"
	// The default value serves as an indicator for timer still running even after MaxStartupTime. Used in case
	// a plugin takes long time to load or is stuck.
	defaultTimerValue = -1
)

// init result flags
const (
	done      = "done"
	errStatus = "error"
	timeout   = "timeout"
)

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
	plugins := flavor.Plugins()

	var agentCoreLogger logging.Logger
	maxStartup := 15 * time.Second

	var flavors []Flavor
	if fs, ok := flavor.(flavorAggregator); ok {
		flavors = fs.fs
	} else {
		flavors = []Flavor{flavor}
	}

	flavor.Inject()

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

	a := Agent{
		plugins,
		agentCoreLogger,
		"",
		Timer{
			MaxStartupTime: maxStartup,
			init:           defaultTimerValue,
			afterInit:      defaultTimerValue,
		},
	}
	return &a
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
	a := Agent{
		plugins,
		logger,
		"",
		Timer{
			MaxStartupTime: maxStartup,
			init:           defaultTimerValue,
			afterInit:      defaultTimerValue,
		},
	}
	return &a
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

	doneChannel := make(chan struct{}, 0)
	errChannel := make(chan error, 0)

	if !flag.Parsed() {
		flag.Parse()
	}

	go func() {
		agent.timer.agentStart = time.Now()
		err := agent.initPlugins()
		if err != nil {
			errChannel <- err
			return
		}
		err = agent.handleAfterInit()
		if err != nil {
			errChannel <- err
			return
		}
		close(doneChannel)
	}()

	// Block until all Plugins are initialized or timeout expires.
	select {
	case err := <-errChannel:

		agent.printStatistics(errStatus)
		return err
	case <-doneChannel:
		agent.printStatistics(done)
		return nil
	case <-time.After(agent.timer.MaxStartupTime):
		agent.printStatistics(timeout)
		return fmt.Errorf(logTimeoutFmt, agent.currentlyProcessing)
	}
}

// Stop gracefully shuts down the Agent. It is called usually when the user
// interrupts the Agent from the EventLoopWithInterrupt().
//
// This implementation tries to call Close() method on every plugin on the list
// in the reverse order. It continues even if some error occurred.
func (agent *Agent) Stop() error {
	agent.Info("Stopping agent...")
	errMsg := ""
	for i := len(agent.plugins) - 1; i >= 0; i-- {
		agent.WithField("pluginName", agent.plugins[i].PluginName).Debug("Stopping plugin begin")
		err := safeclose.Close(agent.plugins[i].Plugin)
		if err != nil {
			if len(errMsg) > 0 {
				errMsg += "; "
			}
			errMsg += string(agent.plugins[i].PluginName)
			errMsg += ": " + err.Error()
		}
		agent.WithField("pluginName", agent.plugins[i].PluginName).Debug("Stopping plugin end ", err)
	}

	agent.Debug("Agent stopped")

	if len(errMsg) > 0 {
		return errors.New(errMsg)
	}
	return nil
}

// initPlugins calls Init() on all plugins in the list.
func (agent *Agent) initPlugins() error {
	// Flag indicates that some of the plugins failed to initialize
	var initPluginCounter int
	var pluginFailed bool
	var wasError error

	// agent.initDuration = defaultTimerValue
	agent.timer.initStart = time.Now()
	for index, plugin := range agent.plugins {
		initPluginCounter = index

		// Set currently initialized plugin name.
		agent.currentlyProcessing = string(plugin.PluginName)

		// Skip all other plugins if some of them failed.
		if pluginFailed {
			agent.Info(fmt.Sprintf(logSkippedFmt, plugin.PluginName))
			continue
		}

		pluginStartTime := time.Now()
		err := plugin.Init()
		if err != nil {
			pluginErrTime := time.Since(pluginStartTime)
			agent.WithField("durationInNs", pluginErrTime.Nanoseconds()).Errorf(logErrorFmt, plugin.PluginName, err, pluginErrTime)

			pluginFailed = true
			wasError = fmt.Errorf(logErrorFmt, plugin.PluginName, err, pluginErrTime)
		} else {
			pluginSuccTime := time.Since(pluginStartTime)
			agent.WithField("durationInNs", pluginSuccTime.Nanoseconds()).Infof(logSuccessFmt, plugin.PluginName, pluginSuccTime)
		}
	}
	agent.timer.init = time.Since(agent.timer.initStart)

	if wasError != nil {
		//Stop the plugins that are initialized
		for i := initPluginCounter; i >= 0; i-- {
			agent.Debugf("Closing %v", agent.plugins[i])
			err := safeclose.Close(agent.plugins[i])
			if err != nil {
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
	var pluginFailed bool
	var wasError error

	// agent.afterInitDuration = defaultTimerValue
	agent.timer.afterInitStart = time.Now()
	for _, plug := range agent.plugins {
		// Set currently after-initialized plugin name.
		agent.currentlyProcessing = string(plug.PluginName)

		// Skip all other plugins if some of them failed.
		if pluginFailed {
			agent.Info(fmt.Sprintf(logAfterSkippedFmt, plug.PluginName))
			continue
		}

		// Check if plugin implements AfterInit().
		if plugin, ok := plug.Plugin.(PostInit); ok {
			pluginStartTime := time.Now()
			err := plugin.AfterInit()
			if err != nil {
				pluginErrTime := time.Since(pluginStartTime)
				agent.WithField("durationInNs", pluginErrTime.Nanoseconds()).Errorf(logAfterErrorFmt, plug.PluginName, err, pluginErrTime)

				pluginFailed = true
				wasError = fmt.Errorf(logAfterErrorFmt, plug.PluginName, err, pluginErrTime)
			} else {
				pluginSuccTime := time.Since(pluginStartTime)
				agent.WithField("durationInNs", pluginSuccTime.Nanoseconds()).Infof(logAfterSuccessFmt, plug.PluginName, pluginSuccTime)
			}
		} else {
			agent.Info(fmt.Sprintf(logNoAfterInitFmt, plug.PluginName))
		}
	}
	agent.timer.afterInit = time.Since(agent.timer.afterInitStart)

	if wasError != nil {
		agent.Stop()
		return wasError
	}
	return nil
}

// Print detailed log entry about startup time.
func (agent *Agent) printStatistics(flag string) {
	switch flag {
	case done:
		overall := agent.timer.init + agent.timer.afterInit
		agent.WithField("durationInNs", overall.Nanoseconds()).Info(fmt.Sprintf("All plugins initialized successfully, took %v", overall))
		agent.WithField("durationInNs", agent.timer.init.Nanoseconds()).Infof("Agent Init took %v", agent.timer.init)
		agent.WithField("durationInNs", agent.timer.afterInit.Nanoseconds()).Infof("Agent AfterInit took %v", agent.timer.afterInit)
	case errStatus:
		agent.WithField("durationInNs", agent.timer.init.Nanoseconds()).Infof("Agent Init took %v", agent.timer.init)
		agent.WithField("durationInNs", agent.timer.afterInit.Nanoseconds()).Infof("Agent AfterInit took %v", agent.timer.afterInit)
	case timeout:
		if agent.timer.init == defaultTimerValue {
			agent.Infof("Agent Init took > %v", agent.timer.MaxStartupTime)
			agent.WithField("durationInNs", agent.timer.afterInit.Nanoseconds()).Infof("Agent AfterInit took %v", agent.timer.afterInit)
		} else if agent.timer.afterInit == defaultTimerValue {
			agent.WithField("durationInNs", agent.timer.init.Nanoseconds()).Infof("Agent Init took %v", agent.timer.init)
			agent.Infof("Agent AfterInit took > %v", agent.timer.MaxStartupTime)
		}
	}

}
