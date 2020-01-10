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

package artifact

type SaveBehavior int32

const (
	SaveAsDir SaveBehavior = 2 << iota
	SaveAsArchive
	PrintToConsole
)

type Config interface {
	SaveBehavior() SaveBehavior
	OutputPath() string
	SaveInAnyCase() bool
}

func ConfigFromEnv() Config {
	var b SaveBehavior
	if processInToConsole.GetBooleanOrDefault(false) {
		b |= PrintToConsole
	}
	if processInToArchive.GetBooleanOrDefault(false) {
		b |= SaveAsArchive
	}
	if processInToDir.GetBooleanOrDefault(false) {
		b |= SaveAsDir
	}
	dir := dir.GetStringOrDefault(defaultOutputPath)
	saveInAnyCase := processInAnyCase.GetBooleanOrDefault(false)
	return &config{
		behavior:      b,
		outputPath:    dir,
		saveInAnyCase: saveInAnyCase,
	}
}

type config struct {
	behavior      SaveBehavior
	outputPath    string
	saveInAnyCase bool
}

func (c *config) SaveInAnyCase() bool {
	return c.saveInAnyCase
}

func (c *config) OutputPath() string {
	return c.outputPath
}

func (c *config) SaveBehavior() SaveBehavior {
	return c.behavior
}
