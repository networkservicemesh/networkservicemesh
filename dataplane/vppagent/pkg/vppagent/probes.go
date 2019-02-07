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

package vppagent

import (
	"fmt"
	"net/http"
)

const (
	healthcheckProbesPort = "0.0.0.0:5555"
)

var (
	srcIPOK        = true
	validIPOK      = true
	ifNameOK       = true
	socketCleanOK  = true
	socketListenOK = true
)

func SetSrcIPFailed() {
	srcIPOK = false
}

func SetValidIPFailed() {
	validIPOK = false
}

func SetExtractIFNameFailed() {
	ifNameOK = false
}

func SetSocketCleanFailed() {
	socketCleanOK = false
}

func SetSocketListenFailed() {
	socketListenOK = false
}

func readiness(w http.ResponseWriter, r *http.Request) {
	if !srcIPOK || !validIPOK || !ifNameOK || !socketCleanOK || !socketListenOK {
		errMsg := fmt.Sprintf("VPP Agent not ready. srcIPOK - %t, validIPOK - %t, ifNameOK - %t, socketCleanOK - %t, socketListenOK - %t", srcIPOK, validIPOK, ifNameOK, socketCleanOK, socketListenOK)
		http.Error(w, errMsg, http.StatusServiceUnavailable)
	} else {
		w.Write([]byte("OK"))
	}
}

func liveness(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func BeginHealthCheck() {
	http.HandleFunc("/liveness", liveness)
	http.HandleFunc("/readiness", readiness)
	http.ListenAndServe(healthcheckProbesPort, nil)
}
