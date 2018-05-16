// Copyright (c) 2017 Cisco and/or its affiliates.
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

// Package probe implements Liveness/Readiness/Prometheus health/metrics HTTP handlers.
package probe

import (
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/health/statuscheck"
	"github.com/ligato/cn-infra/rpc/prometheus"
	"github.com/ligato/cn-infra/rpc/rest"
)

// Deps lists dependencies of REST plugin.
type Deps struct {
	local.PluginInfraDeps                          // inject
	HTTP                  rest.HTTPHandlers        // inject
	StatusCheck           statuscheck.StatusReader // inject
}

// PrometheusDeps lists dependencies of Prometheus plugin.
type PrometheusDeps struct {
	local.PluginInfraDeps                          // inject
	HTTP                  rest.HTTPHandlers        // inject
	StatusCheck           statuscheck.StatusReader // inject
	Prometheus            prometheus.API           // inject
}
