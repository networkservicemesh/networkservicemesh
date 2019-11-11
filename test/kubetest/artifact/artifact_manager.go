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

import "sync"

type Manager interface {
	ProcessArtifacts()
}

func NewManager(c Config, factory PresenterFactory, finders []Finder, hooks []Hook) Manager {
	return &manager{
		presenter: factory.Presenter(c),
		finders:   finders,
		hooks:     append(hooks, factory.Hooks(c)...),
	}
}

type manager struct {
	finders   []Finder
	hooks     []Hook
	presenter Presenter
}

func (m *manager) ProcessArtifacts() {
	for _, hook := range m.hooks {
		hook.Started()
	}
	wg := sync.WaitGroup{}
	artifactCh := make(chan Artifact, artifactWorkerCount)
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
	for i := 0; i < artifactWorkerCount; i++ {
		go func() {
			for artifact := range artifactCh {
				for _, hook := range m.hooks {
					artifact = hook.PreProcess(artifact)
				}
				if artifact == nil {
					wg.Done()
					continue
				}
				m.presenter.Present(artifact)
				for _, hook := range m.hooks {
					hook.PostProcess(artifact)
				}
				wg.Done()
			}
		}()
	}
	wg.Wait()
	close(artifactCh)
	for _, hook := range m.hooks {
		hook.Finished()
	}
}
