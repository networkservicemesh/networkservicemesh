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

package common

import (
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

const (
	healthcheckProbesPort = "0.0.0.0:5555"
)

// DataplaneProbes - Dataplane readiness probes
type DataplaneProbes struct {
	srcIPOK        bool
	validIPOK      bool
	egressOK       bool
	socketCleanOK  bool
	socketListenOK bool
}

// NewDataplaneProbes create a new dataplane readness probes
func NewDataplaneProbes() *DataplaneProbes {
	return &DataplaneProbes{
		false,
		false,
		false,
		false,
		false,
	}
}

// SetSrcIPReady notifies that Dataplane Src IP is configured
func (probes *DataplaneProbes) SetSrcIPReady() {
	probes.srcIPOK = true
}

// SetValidIPReady notifies that configured Dataplane IP is valid
func (probes *DataplaneProbes) SetValidIPReady() {
	probes.validIPOK = true
}

// SetNewEgressIFReady notifies that Egress Interface is found
func (probes *DataplaneProbes) SetNewEgressIFReady() {
	probes.egressOK = true
}

// SetSocketCleanReady notifies that socket is cleaned up
func (probes *DataplaneProbes) SetSocketCleanReady() {
	probes.socketCleanOK = true
}

// SetSocketListenReady notifies that listening on socket is started
func (probes *DataplaneProbes) SetSocketListenReady() {
	probes.socketListenOK = true
}

func (probes *DataplaneProbes) readiness(w http.ResponseWriter, r *http.Request) {
	if !probes.srcIPOK || !probes.validIPOK || !probes.egressOK || !probes.socketCleanOK || !probes.socketListenOK {
		errMsg := fmt.Sprintf("VPP Agent not ready. srcIPOK - %t, validIPOK - %t, egressOK - %t, socketCleanOK - %t, socketListenOK - %t", probes.srcIPOK, probes.validIPOK, probes.egressOK, probes.socketCleanOK, probes.socketListenOK)
		http.Error(w, errMsg, http.StatusServiceUnavailable)
	} else {
		w.Write([]byte("OK"))
	}
}

func (probes *DataplaneProbes) liveness(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// BeginHealthCheck starts listening 5555 port for health check
func (probes *DataplaneProbes) BeginHealthCheck() {
	logrus.Debug("Starting VPP Agent liveness/readiness healthcheck")
	http.HandleFunc("/liveness", probes.liveness)
	http.HandleFunc("/readiness", probes.readiness)
	http.ListenAndServe(healthcheckProbesPort, nil)
}
