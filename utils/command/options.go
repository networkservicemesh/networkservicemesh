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

package command

import (
	"sync"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "setme"}

var commandsByName sync.Map

// RootCmd returns the rootCmd previously set by SetRootCmd.RootCmd
// If RootCmd() returns nil, no root command has been yet set
func RootCmd() *cobra.Command {
	return rootCmd
}

// SetRootCmd sets the root command to the provided cmd
func SetRootCmd(cmd *cobra.Command) {
	rootCmd = cmd
}

// CmdByName allows the retrieval of a shared cmd by name
// returns nil of no command by that name has been shared
func CmdByName(name string) *cobra.Command {
	result, ok := commandsByName.Load(name)
	if ok {
		return result.(*cobra.Command)
	}
	return nil
}

// SetCmdByName stores a cmd by name so others can retrieve it
func SetCmdByName(name string, cmdByName *cobra.Command) {
	commandsByName.Store(name, cmdByName)
}
