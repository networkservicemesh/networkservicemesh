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
	"fmt"
	"runtime"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// startMetrics starts the metrics monitoring
func (v *KernelForwarder) startMetrics() {
	logrus.Info("Starting metrics collector...")
	go serveMetrics(v.common.Monitor, v.common.MetricsPeriod, v.devices)
}

// serveMetrics is started as a routine
func serveMetrics(monitor metrics.MetricsMonitor, requestPeriod time.Duration, devices *registeredDevices) {
	for {
		devices.Lock()
		/* Collect metrics for each interface */
		stats, err := collectMetrics(devices.devices)
		devices.Unlock()
		if err != nil {
			logrus.Error("failed to collect metrics:", err)
		}
		/* Send metrics update */
		monitor.HandleMetrics(stats)
		/* Wait until next check */
		time.Sleep(requestPeriod)
	}
}

func collectMetrics(devices map[string]string) (map[string]*crossconnect.Metrics, error) {
	logrus.Info("----- collecting metrics -----")
	/* Store the metrics for all registered devices here */
	stats := make(map[string]*crossconnect.Metrics)
	/* Loop through each registered device */
	for device, namespace := range devices {
		metrics, err := getDeviceMetrics(device, namespace)
		if err != nil {
			logrus.Errorf("failed to extract metrics for device %s in namespace %s: %v", device, namespace, err)
		}
		logrus.Info("-------- device, metrics", device, metrics)
		stats[device] = &crossconnect.Metrics{Metrics: metrics}
	}
	logrus.Info("------- all stats collected are", stats)
	return stats, nil
}

// getDeviceMetrics returns metrics for device in specific namespace
func getDeviceMetrics(device, namespace string) (map[string]string, error) {
	logrus.Info("------ 1. Extracting metrics for device in namespace ", device, namespace)
	/* 1. Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	/* 2. Save the current network namespace */
	currentNs, err := netns.Get()
	defer func() {
		if err = currentNs.Close(); err != nil {
			logrus.Error("error when closing current namespace:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("failed to get current namespace: %v", err)
		return nil, err
	}
	logrus.Info("------ 2. current namespace got -----")
	/* 3. Get handler for namespace */
	nsHandle, err := netns.GetFromPath(namespace)
	defer func() {
		if err = nsHandle.Close(); err != nil {
			logrus.Error("error when closing desired namespace:", err)
		}
	}()
	if err != nil {
		logrus.Errorf("failed to get namespace handler from path - %v", err)
		return nil, err
	}
	logrus.Info("------ 3. desired namespace got -----")
	/* 4. Switch to the new namespace */
	if err = netns.Set(nsHandle); err != nil {
		logrus.Errorf("failed to switch to container namespace: %v", err)
		return nil, err
	}
	logrus.Info("------ 4. switched to desired namespace -----")
	/* 5. Get a link for the interface name */
	link, err := netlink.LinkByName(device)
	if err != nil {
		logrus.Errorf("failed to lookup %q, %v", device, err)
		return nil, err
	}
	logrus.Info("------ 5. got interface link -----")
	/* 6. Save the statistics in a separate metrics map */
	metrics := make(map[string]string)
	metrics["rx_bytes"] = fmt.Sprint(link.Attrs().Statistics.RxBytes)
	metrics["tx_bytes"] = fmt.Sprint(link.Attrs().Statistics.TxBytes)
	metrics["rx_packets"] = fmt.Sprint(link.Attrs().Statistics.RxPackets)
	metrics["tx_packets"] = fmt.Sprint(link.Attrs().Statistics.TxPackets)
	metrics["rx_error_packets"] = fmt.Sprint(link.Attrs().Statistics.RxErrors)
	metrics["tx_error_packets"] = fmt.Sprint(link.Attrs().Statistics.TxErrors)

	/* 7. Switch back to the original namespace */
	if err = netns.Set(currentNs); err != nil {
		logrus.Errorf("failed to switch back to original namespace: %v", err)
		return nil, err
	}
	logrus.Info("------ 6. switched to original namespace -----")
	return metrics, nil
}
