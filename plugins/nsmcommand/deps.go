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

package nsmcommand

import (
	"github.com/ligato/networkservicemesh/plugins/crd"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/nsmserver"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"github.com/spf13/cobra"
)

// Deps - dependencies for Plugin
type Deps struct {
	Name        string
	Log         logger.FieldLoggerPlugin
	Cmd         *cobra.Command
	NSMServer   nsmserver.PluginAPI
	CRD         netmeshplugincrd.PluginAPI
	ObjectStore objectstore.PluginAPI
}
