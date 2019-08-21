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

package kernelforwarder

import (
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"
	"github.com/sirupsen/logrus"
)

// startMetrics starts the metrics monitoring
func (v *KernelForwarder) startMetrics() {
	logrus.Info("Starting metrics collector...")
	go serveMetrics(v.common.Monitor, v.common.MetricsPeriod)
}

// serveMetrics is started as a routine
func serveMetrics(monitor metrics.MetricsMonitor, requestPeriod time.Duration) {
	for {
		/* Collect metrics for each interface */
		stats, err := collectMetrics()
		if err != nil {
			logrus.Error("failed to collect metrics:", err)
		}
		/* Send metrics update */
		monitor.HandleMetrics(stats)
		/* Wait until next check */
		time.Sleep(requestPeriod)
	}
}

func collectMetrics() (map[string]*crossconnect.Metrics, error) {
	name := "foo-interface"
	metrics := make(map[string]string)
	metrics["rx_bytes"] = "1111"
	metrics["tx_bytes"] = "2222"
	metrics["rx_packets"] = "3333"
	metrics["tx_packets"] = "4444"
	metrics["rx_error_packets"] = "5555"
	metrics["tx_error_packets"] = "6666"
	stats := map[string]*crossconnect.Metrics{
		name: {Metrics: metrics},
	}
	return stats, nil
}
