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

package kubetest

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/artifacts"
)

type k8sLogFinder struct {
	k8s *K8s
}

//NewK8sLogFinder creates finder for pods logs
func NewK8sLogFinder(k8s *K8s) artifacts.Finder {
	return &k8sLogFinder{k8s: k8s}
}

func (k *k8sLogFinder) Find() []artifacts.Artifact {
	pods, err := k.k8s.ListPods()
	if err != nil {
		logrus.Errorf("Can not find k8s artifacts: %v", err.Error())
		return nil
	}
	ch := make(chan *v1.Pod, artifactFinderWorkerCount)
	wg := sync.WaitGroup{}
	wg.Add(1)
	var result []artifacts.Artifact
	var m sync.Mutex
	addArtifact := func(a artifacts.Artifact) {
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
	addArtifactFromContainers := func(p *v1.Pod, containers []v1.Container, prev bool) {
		for i := 0; i < len(containers); i++ {
			c := &containers[i]
			logs, err := k.k8s.GetLogsWithOptions(p, &v1.PodLogOptions{
				Previous:  prev,
				Container: c.Name,
				SinceTime: &k.k8s.startTime,
			})
			if err != nil {
				logrus.Warningf("Can not get logs for container: %v, Error: %v", c.Name, err)
				return
			}
			addArtifact(artifacts.New(nameForArtifact(p, c, prev), "log", []byte(logs)))
		}
	}
	for i := 0; i < artifactFinderWorkerCount; i++ {
		go func() {
			for p := range ch {
				for _, prev := range []bool{false, true} {
					addArtifactFromContainers(p, p.Spec.Containers, prev)
					addArtifactFromContainers(p, p.Spec.InitContainers, prev)
				}
				wg.Done()
			}
		}()
	}
	wg.Wait()
	close(ch)
	logrus.Infof("artifacts: %+v", result)
	return result
}

func nameForArtifact(p *v1.Pod, c *v1.Container, prev bool) string {
	name := fmt.Sprintf("%v-%v.log", p.Name, c.Name)
	if prev {
		name = "previous-" + name
	}
	return name
}
