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

package monitoring

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// Metrics monitoring instance type
type Metrics struct {
	requestPeriod time.Duration
	devices       *RegisteredDevices
}

// RegisteredDevices keeps track of all devices created by the forwarding plane
type RegisteredDevices struct {
	sync.Mutex
	devices map[string]string
}

// CreateMetricsMonitor creates new metric monitoring instance
func CreateMetricsMonitor(requestPeriod time.Duration) *Metrics {
	return &Metrics{
		requestPeriod: requestPeriod,
		devices:       &RegisteredDevices{devices: map[string]string{}},
	}
}

// GetDevices returns the currently available devices
func (m *Metrics) GetDevices() *RegisteredDevices {
	return m.devices
}

// Start starts the monitoring
func (m *Metrics) Start(monitor metrics.MetricsMonitor) {
	logrus.Info("metrics: monitoring started")
	go serveMetrics(monitor, m.requestPeriod, m.devices)
}

// serveMetrics aims to be started as a Go routine
func serveMetrics(monitor metrics.MetricsMonitor, requestPeriod time.Duration, devices *RegisteredDevices) {
	for {
		if len(devices.devices) != 0 {
			/* Collect metrics for all present devices */
			stats, err := collectMetrics(devices)
			if err != nil {
				logrus.Error("metrics: failed to collect metrics:", err)
			}
			/* Send metrics update */
			monitor.HandleMetrics(stats)
		}
		/* Wait until next check */
		time.Sleep(requestPeriod)
	}
}

// collectMetrics loops over each device and extracts the metrics for it
func collectMetrics(devices *RegisteredDevices) (map[string]*crossconnect.Metrics, error) {
	/* Store the metrics for all registered devices here */
	stats := make(map[string]*crossconnect.Metrics)
	var failedDevices map[string]string
	devices.Lock()
	/* Loop through each registered device */
	for device, namespace := range devices.devices {
		metrics, err := getDeviceMetrics(device, namespace)
		if err != nil {
			logrus.Errorf("metrics: failed to extract metrics for device %s in namespace %s: %v", device, namespace, err)
			logrus.Errorf("metrics: removing device %s from device list", device)
			failedDevices[device] = ""
		} else {
			logrus.Infof("metrics: device %s has the following metrics %v", device, metrics)
			stats[device] = &crossconnect.Metrics{Metrics: metrics}
		}
	}
	devices.Unlock()
	if failedDevices != nil {
		devices.UpdateDeviceList(failedDevices, false)
	}
	return stats, nil
}

// getDeviceMetrics returns metrics for device in specific namespace
func getDeviceMetrics(device, namespace string) (map[string]string, error) {
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
	/* 4. Switch to the new namespace */
	if err = netns.Set(nsHandle); err != nil {
		logrus.Errorf("failed to switch to container namespace: %v", err)
		return nil, err
	}
	/* 5. Get a link for the interface name */
	link, err := netlink.LinkByName(device)
	if err != nil {
		logrus.Errorf("failed to lookup %q, %v", device, err)
		return nil, err
	}
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
	return metrics, nil
}

// UpdateDeviceList keeps track of the devices being handled by the Kernel forwarding plane
func (m *RegisteredDevices) UpdateDeviceList(devices map[string]string, connect bool) error {
	/* Add devices */
	m.Lock()
	defer m.Unlock()
	if connect {
		for device, namespace := range devices {
			_, ok := m.devices[device]
			if ok {
				logrus.Error("metrics: device requested for add is already present in the devices list")
				return fmt.Errorf("metrics: device requested for add is already present in the devices list")
			}
			m.devices[device] = namespace
		}
	} else {
		/* Delete devices */
		for device := range devices {
			_, ok := m.devices[device]
			if !ok {
				logrus.Error("metrics: device requested for delete is already missing from the devices list")
				return fmt.Errorf("metrics: device requested for delete is already missing from the devices list")
			}
			delete(m.devices, device)
		}
	}
	logrus.Infof("metrics: device list - %v", m.devices)
	return nil
}
