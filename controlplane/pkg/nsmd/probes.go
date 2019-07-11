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

package nsmd

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

const (
	healthcheckProbesPort = "0.0.0.0:5555"
)

// Probes - Network Service Manager readiness probes
type Probes struct {
	dpStatusOK       bool
	nsmStatusOK      bool
	apiStatusOK      bool
	listenerStatusOK bool
}

// NewProbes creates new Network Service Manager readiness probes
func NewProbes() *Probes {
	return &Probes{}
}

// SetDPServerReady notifies that dataplane is available
func (probes *Probes) SetDPServerReady() {
	probes.dpStatusOK = true
}

// SetNSMServerReady notifies that NSM Server is started
func (probes *Probes) SetNSMServerReady() {
	probes.nsmStatusOK = true
}

// SetAPIServerReady notifies that API Server is started
func (probes *Probes) SetAPIServerReady() {
	probes.apiStatusOK = true
}

// SetPublicListenerReady notifies that Public API server is started
func (probes *Probes) SetPublicListenerReady() {
	probes.listenerStatusOK = true
}

func (probes *Probes) readiness(w http.ResponseWriter, r *http.Request) {
	if !probes.dpStatusOK || !probes.nsmStatusOK || !probes.apiStatusOK || !probes.listenerStatusOK {
		errMsg := fmt.Sprintf("NSMD not ready. DPServer - %t, NSMServer - %t, APIServer - %t, PublicListener - %t", probes.dpStatusOK, probes.nsmStatusOK, probes.apiStatusOK, probes.listenerStatusOK)
		http.Error(w, errMsg, http.StatusServiceUnavailable)
	} else {
		w.Write([]byte("OK"))
	}
}

func (probes *Probes) liveness(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// BeginHealthCheck starts listening 5555 port for health check
func (probes *Probes) BeginHealthCheck() {
	logrus.Debug("Starting NSMD liveness/readiness healthcheck")
	http.HandleFunc("/liveness", probes.liveness)
	http.HandleFunc("/readiness", probes.readiness)
	http.ListenAndServe(healthcheckProbesPort, nil)
}
