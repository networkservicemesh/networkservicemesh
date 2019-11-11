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
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/artifact"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/jaeger"
)

type jaegerTracesFinder struct {
	k8s *K8s
}

func NewJaegerTracesFinder(k8s *K8s) artifact.Finder {
	return &jaegerTracesFinder{k8s: k8s}
}

func (j *jaegerTracesFinder) Find() []artifact.Artifact {
	var result []artifact.Artifact
	var m sync.Mutex
	addArtifact := func(a artifact.Artifact) {
		m.Lock()
		result = append(result, a)
		m.Unlock()
	}
	jaegerPod := FindJaegerPod(j.k8s)
	if jaegerPod == nil {
		logrus.Warn("Jaeger pod not found. Traces not saved.")
		return result
	}
	fwd, err := j.k8s.NewPortForwarder(jaegerPod, jaeger.GetRestAPIPort())
	if err != nil {
		logrus.Errorf("Can not create port forwarder for pod: %v. Error: %v", jaegerPod.Name, err)
		return result
	}
	err = fwd.Start()
	if err != nil {
		logrus.Errorf("Can not start port forwarder for pod: %v. Error: %v", jaegerPod.Name, err)
		return result
	}
	defer fwd.Stop()
	c := &jaegerAPIClient{
		apiServerPort: fwd.ListenPort,
	}
	servicesCh := make(chan string, artifactFinderWorkerCount)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, s := range c.getServices() {
			wg.Add(1)
			servicesCh <- s
		}

	}()
	for i := 0; i < artifactFinderWorkerCount; i++ {
		go func() {
			for service := range servicesCh {
				traces := c.getTracesByService(service)
				addArtifact(artifact.New(service+".json", "json", []byte(traces)))
				wg.Done()
			}
		}()
	}
	wg.Wait()
	close(servicesCh)
	return result
}
