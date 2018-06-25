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

// Simplified wrapper around github.com/ligato/cn-infra/logging that supports Sharing of Loggers
package log

import (
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/logrus"
	"sync"
)

// Unless told  otherwise, lets use a defaultRegistry
var defaultLogRegistry = logrus.NewLogRegistry().(logging.LogFactory)

// We have to track the Loggers because asking for a new one with the same name
// will lead to a panic
var logMap sync.Map

// log simply implements the Logger interface by wrapping PluginLogger and adding Name()
type log struct {
	logging.PluginLogger
}

// Name returns the name of the plugin
func (l *log) Name() string {
	return l.GetName()
}

// SharedLog will retrieve a PluginLogger or create it if it doesn't exist
// name - name for logger
// parent - parent plugin if creating the log for a child Plugin
//          if parent is not supplied, will use a defaultLogRegistry
func SharedLog(name string, parent ...Logger) Logger {
	logger := &log{}
	n := name // need a copy of name in case we have to add a prefix to it
	factory := defaultLogRegistry

	if len(parent) > 0 {
		factory = parent[0]
		n = parent[0].GetName() + n // Have to compensate for use of parent as prefix
	}

	result, found := logMap.LoadOrStore(n, logger)
	if found {
		logger, _ = result.(*log)
		return logger
	}

	logger.PluginLogger = logging.ForPlugin(name, factory)
	return logger
}
