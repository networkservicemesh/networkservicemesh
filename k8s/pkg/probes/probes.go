// Copyright 2019 VMware, Inc.
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

package probes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/probes/health"

	"github.com/sirupsen/logrus"
)

const (
	healthcheckProbesPort = "0.0.0.0:5555"
	probeTimeout          = time.Second
)

type Probes interface {
	BeginHealthCheck()
	health.Appender
}

// probesImpl - Network Service Manager readiness probes
type probesImpl struct {
	name  string
	goals Goals
	*health.AppenderImpl
}

// NewProbes creates new Network Service Manager readiness probes
func NewProbes(name string, goals Goals) Probes {
	return &probesImpl{
		name:         name,
		goals:        goals,
		AppenderImpl: new(health.AppenderImpl),
	}
}

func (probes *probesImpl) readiness(w http.ResponseWriter, r *http.Request) {
	if !probes.goals.IsComplete() {
		http.Error(w, probes.goals.Status(), http.StatusServiceUnavailable)
		return
	}
	var err error
	probes.Iterate(func(checker health.ApplicationHealth) bool {
		err = checker.Check()
		return err == nil
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Goals status: %v, health check status: %v", probes.goals.Status(), err), http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("OK"))
}

func (probes *probesImpl) liveness(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// BeginHealthCheck starts listening 5555 port for health check
func (probes *probesImpl) BeginHealthCheck() {
	go func() {
		logrus.Debugf("Starting %v", probes.name)
		http.HandleFunc("/liveness", probes.liveness)
		http.HandleFunc("/readiness", probes.readiness)
		http.ListenAndServe(healthcheckProbesPort, nil)
	}()
}
