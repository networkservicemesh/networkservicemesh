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

package logmanager

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"

	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/rpc/rest"
)

// LoggerData encapsulates parameters of a logger represented as strings.
type LoggerData struct {
	Logger string `json:"logger"`
	Level  string `json:"level"`
}

// Variable names in logger registry URLs
const (
	loggerVarName = "logger"
	levelVarName  = "level"
)

// Plugin allows to manage log levels of the loggers using HTTP.
type Plugin struct {
	Deps
	*Conf
}

// Deps groups dependencies injected into the plugin so that they are
// logically separated from other plugin fields.
type Deps struct {
	Log                 logging.PluginLogger //inject
	PluginName          core.PluginName      //inject
	config.PluginConfig                      //inject

	LogRegistry logging.Registry  // inject
	HTTP        rest.HTTPHandlers // inject
}

// NewConf creates default configuration with InfoLevel & empty loggers.
// Suitable also for usage in flavor to programmatically specify default behavior.
func NewConf() *Conf {
	return &Conf{
		DefaultLevel: "",
		Loggers:      []ConfLogger{},
	}
}

// Conf is a binding that supports to define default log levels for multiple loggers
type Conf struct {
	DefaultLevel string
	Loggers      []ConfLogger
}

// ConfLogger is configuration of a particular logger.
// Currently we support only logger level.
type ConfLogger struct {
	Name  string
	Level string //debug, info, warning, error, fatal, panic
}

// Init does nothing
func (lm *Plugin) Init() error {
	if lm.PluginConfig != nil {
		if lm.Conf == nil {
			lm.Conf = NewConf()
		}

		_, err := lm.PluginConfig.GetValue(lm.Conf)
		if err != nil {
			return err
		}
		lm.Log.Debugf("logs config: %+v", lm.Conf)

		if lm.Conf.DefaultLevel != "" {
			if err := lm.LogRegistry.SetLevel("default", lm.Conf.DefaultLevel); err != nil {
				lm.Log.Warn("setting default log level failed:", err)
			} else {
				lm.Log.Debugf("default log level to %q", lm.Conf.DefaultLevel)
			}
		}

		// try to set log levels (note, not all of them might exist yet)
		for _, cfgLogger := range lm.Conf.Loggers {
			if err := lm.LogRegistry.SetLevel(cfgLogger.Name, cfgLogger.Level); err != nil {
				//intentionally just log warn & not propagate the error (it is minor thing to interrupt startup)
				lm.Log.Warn("setting level failed:", err)
			}
		}

	}

	return nil
}

// AfterInit is called at plugin initialization. It register the following handlers:
// - List all registered loggers:
//   > curl -X GET http://localhost:<port>/log/list
// - Set log level for a registered logger:
//   > curl -X PUT http://localhost:<port>/log/<logger-name>/<log-level>
func (lm *Plugin) AfterInit() error {
	if lm.HTTP != nil {
		lm.HTTP.RegisterHTTPHandler(fmt.Sprintf("/log/{%s}/{%s:debug|info|warning|error|fatal|panic}",
			loggerVarName, levelVarName), lm.logLevelHandler, "PUT")
		lm.HTTP.RegisterHTTPHandler("/log/list", lm.listLoggersHandler, "GET")
	}
	return nil
}

// Close is called at plugin cleanup phase.
func (lm *Plugin) Close() error {
	return nil
}

// ListLoggers lists all registered loggers.
func (lm *Plugin) listLoggers() []LoggerData {
	loggers := []LoggerData{}

	lgs := lm.LogRegistry.ListLoggers()
	for lg, lvl := range lgs {
		ld := LoggerData{
			Logger: lg,
			Level:  lvl,
		}
		loggers = append(loggers, ld)
	}

	return loggers
}

// setLoggerLogLevel modifies the log level of the all loggers in a plugin
func (lm *Plugin) setLoggerLogLevel(name string, level string) error {
	lm.Log.Debugf("SetLogLevel name '%s', level '%s'", name, level)

	return lm.LogRegistry.SetLevel(name, level)
}

// logLevelHandler processes requests to set log level on loggers in a plugin
func (lm *Plugin) logLevelHandler(formatter *render.Render) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {
		lm.Log.Infof("Path: %s", req.URL.Path)
		vars := mux.Vars(req)
		if vars == nil {
			formatter.JSON(w, http.StatusNotFound, struct{}{})
			return
		}
		err := lm.setLoggerLogLevel(vars[loggerVarName], vars[levelVarName])
		if err != nil {
			formatter.JSON(w, http.StatusNotFound,
				struct{ Error string }{err.Error()})
			return
		}
		formatter.JSON(w, http.StatusOK,
			LoggerData{Logger: vars[loggerVarName], Level: vars[levelVarName]})
	}
}

// listLoggersHandler processes requests to list all registered loggers
func (lm *Plugin) listLoggersHandler(formatter *render.Render) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		formatter.JSON(w, http.StatusOK, lm.listLoggers())
	}
}
