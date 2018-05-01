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

package connectors

import (
	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/datasync/kvdbsync"
	"github.com/ligato/cn-infra/datasync/resync"
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/ligato/cn-infra/flavors/local"
)

// InjectKVDBSync sets kvdbsync.Plugin dependencies.
// The intent of this method is just extract code that would be copy&pasted otherwise.
func InjectKVDBSync(dbsync *kvdbsync.Plugin,
	db keyval.KvProtoPlugin, dbPlugName core.PluginName, local *local.FlavorLocal, resync resync.Subscriber) {

	dbsync.Deps.PluginLogDeps = *local.LogDeps(string(dbPlugName) + "-datasync")
	dbsync.KvPlugin = db
	dbsync.ResyncOrch = resync
	if local != nil {
		//Note, not injecting local.ETCDDataSync.ResyncOrch here

		dbsync.ServiceLabel = &local.ServiceLabel

		if local.StatusCheck.Transport == nil {
			local.StatusCheck.Transport = dbsync
		}
	}
}
