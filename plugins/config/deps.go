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

package config

import (
	"github.com/spf13/cobra"
)

// Deps is the dependencies for config.Plugin
//      Name:          Name of the config.Plugin  Used as the key for the
//                     subsection of the config file from which to load the
//                     config
//      Cmd:           The cobra.Cmd to bind to for flags.  Defaults to
//                     command.RootCmd()
//      DefaultConfig: Default config to use if there is no matching config
//                     in the config files.  Is also used to figure out how
//                     to unmarshal the config
type Deps struct {
	Name          string
	Cmd           *cobra.Command
	DefaultConfig Config
}
