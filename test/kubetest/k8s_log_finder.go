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

package kubetest

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/artifact"
)

type k8sLogFinder struct {
	k8s *K8s
}

func NewK8sLogFinder(k8s *K8s) artifact.Finder {
	return &k8sLogFinder{k8s: k8s}
}

func (k *k8sLogFinder) Find() []artifact.Artifact {
	pods := k.k8s.ListPods()
	ch := make(chan *v1.Pod, artifactFinderWorkerCount)
	wg := sync.WaitGroup{}
	wg.Add(1)
	var result []artifact.Artifact
	var m sync.Mutex
	addArtifact := func(a artifact.Artifact) {
		m.Lock()
		result = append(result, a)
		m.Unlock()
	}
	go func() {
		defer wg.Done()
		for i := range pods {
			wg.Add(1)
			ch <- &pods[i]
		}
	}()
	for i := 0; i < artifactFinderWorkerCount; i++ {
		go func() {
			for p := range ch {
				for _, prev := range []bool{false, true} {
					for j := 0; j < len(p.Spec.Containers); j++ {
						c := &p.Spec.Containers[j]
						logs, err := k.k8s.GetFullLogs(p, c.Name, prev)
						if err != nil {
							logrus.Errorf("Can not get logs for container: %v, Error: %v", c.Name, err)
							continue
						}
						addArtifact(artifact.New(nameForArtifact(p, c, prev), "log", []byte(logs)))
					}
					for j := 0; j < len(p.Spec.InitContainers); j++ {
						c := &p.Spec.InitContainers[j]
						logs, err := k.k8s.GetFullLogs(p, c.Name, prev)
						if err != nil {
							logrus.Errorf("Can not get logs for init container: %v. Error: %v", c.Name, err)
							continue
						}
						addArtifact(artifact.New(nameForArtifact(p, c, prev), "log", []byte(logs)))
					}
				}
				wg.Done()
			}
		}()
	}
	wg.Wait()
	close(ch)
	logrus.Infof("artifacts: %v", result)
	return result
}

func nameForArtifact(p *v1.Pod, c *v1.Container, prev bool) string {
	if prev {
		return fmt.Sprintf("prev-%v-%v.log", p.Name, c.Name)
	}
	return fmt.Sprintf("%v-%v.log", p.Name, c.Name)
}
