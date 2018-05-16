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

package local

import (
	"github.com/ligato/cn-infra/config"
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/servicelabel"
)

// PluginLogDeps is minimal set of plugin dependencies that
// will probably use every plugin to:
// - log messages using plugin logger or child (prefixed) logger (in case plugin
//   needs more than one)
// - to learn the plugin name
type PluginLogDeps struct {
	Log        logging.PluginLogger // inject
	PluginName core.PluginName      // inject
}

// Close is called by Agent Core when the Agent is shutting down.
// It is supposed to clean up resources that were allocated by the plugin
// during its lifetime. This is a default empty implementation used to not bother
// plugins that do not need to implement this method.
func (plugin *PluginLogDeps) Close() error {
	return nil
}

// PluginInfraDeps is a standard set of plugin dependencies that
// will need probably every connector to DB/Messaging:
// - to report/write plugin status to StatusCheck
// - to know micro-service label prefix
type PluginInfraDeps struct {
	PluginLogDeps                                      // inject
	config.PluginConfig                                // inject
	StatusCheck         statuscheck.PluginStatusWriter // inject
	ServiceLabel        servicelabel.ReaderAPI         // inject
}
