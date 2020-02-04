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

import (
	"sync"

	"github.com/sirupsen/logrus"
)

//Manager saves artifacts
type Manager interface {
	SaveArtifacts()
	Config() Config
}

//NewManager creates artifacts manager with specific parameters.
func NewManager(c Config, factory PresenterFactory, finders []Finder, hooks []Hook) Manager {
	logrus.Infof("Creating artifact manager with config: %v", c)
	return &manager{
		presenter: factory.Presenter(c),
		finders:   finders,
		hooks:     append(hooks, factory.Hooks(c)...),
		config:    c,
	}
}

type manager struct {
	finders   []Finder
	hooks     []Hook
	presenter Presenter
	config    Config
}

func (m *manager) Config() Config {
	return m.config
}

func (m *manager) SaveArtifacts() {
	for _, hook := range m.hooks {
		hook.OnStart()
	}
	wg := sync.WaitGroup{}
	artifactCh := make(chan Artifact, saveWorkerCount)
	for i := 0; i < len(m.finders); i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			artifacts := m.finders[index].Find()
			for _, artifact := range artifacts {
				if artifact == nil {
					continue
				}
				wg.Add(1)
				artifactCh <- artifact
			}
		}(i)
	}
	for i := 0; i < saveWorkerCount; i++ {
		go func() {
			for artifact := range artifactCh {
				for _, hook := range m.hooks {
					artifact = hook.OnPresent(artifact)
				}
				if artifact == nil {
					wg.Done()
					continue
				}
				m.presenter.Present(artifact)
				for _, hook := range m.hooks {
					hook.OnPresented(artifact)
				}
				wg.Done()
			}
		}()
	}
	wg.Wait()
	close(artifactCh)
	for _, hook := range m.hooks {
		hook.OnFinish()
	}
}
