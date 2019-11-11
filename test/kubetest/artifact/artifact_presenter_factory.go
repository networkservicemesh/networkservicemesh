// Copyright (c) 2019 Cisco Systems, Inc.
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

type PresenterFactory interface {
	Presenter(config Config) Presenter
	Hooks(config Config) []Hook
}

func DefaultPresenterFactory() PresenterFactory {
	return &defaultFactory{}
}

type defaultFactory struct {
}

func (f *defaultFactory) Hooks(config Config) []Hook {
	return []Hook{Archivator(config)}
}

func (f *defaultFactory) Presenter(config Config) Presenter {
	combined := &combinedPresenter{}
	if config.SaveBehavior()&SaveAsDir == 1 {
		combined.presenters = append(combined.presenters, &filePresenter{path: config.OutputPath()})
	}
	if config.SaveBehavior()&PrintToConsole == 1 {
		combined.presenters = append(combined.presenters, &consolePresnter{})
	}
	return combined
}

type combinedPresenter struct {
	presenters []Presenter
}

func (c *combinedPresenter) Present(a Artifact) {
	for _, p := range c.presenters {
		p.Present(a)
	}
}
