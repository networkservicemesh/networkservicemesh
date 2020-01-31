// Copyright (c) 2019-2020 Cisco Systems, Inc.
//
// SPDX-License-Identifier: Apache-2.0
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

package artifacts

//SaveOption means how artifact will save / presented
type SaveOption byte

const (
	//SaveAsFiles means that artifacts should be saved as files
	SaveAsFiles SaveOption = 2 << iota
	//SaveAsArchive means that artifacts should be saved as archive
	SaveAsArchive
	//PrintToConsole means that artifacts should be printed to console
	PrintToConsole
)

//Config is configuration of artifacts manager
type Config interface {
	SaveOption() SaveOption
	OutputPath() string
	SaveInAnyCase() bool
}

//ConfigFromEnv reads config options from environment variables.
func ConfigFromEnv() Config {
	var b SaveOption
	if printToConsole.GetBooleanOrDefault(false) {
		b |= PrintToConsole
	}
	if archiveArtifacts.GetBooleanOrDefault(false) {
		b |= SaveAsArchive
	}
	if saveAsFiles.GetBooleanOrDefault(false) {
		b |= SaveAsFiles
	}
	dir := outputDirectory.GetStringOrDefault(defaultOutputPath)
	saveInAnyCase := saveInAnyCase.GetBooleanOrDefault(false)
	return &config{
		behavior:      b,
		outputPath:    dir,
		saveInAnyCase: saveInAnyCase,
	}
}

type config struct {
	behavior      SaveOption
	saveInAnyCase bool
	outputPath    string
}

func (c *config) SaveInAnyCase() bool {
	return c.saveInAnyCase
}

func (c *config) OutputPath() string {
	return c.outputPath
}

func (c *config) SaveOption() SaveOption {
	return c.behavior
}
