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

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"
)

// Metrics monitoring instance type
type Metrics struct {
	requestPeriod time.Duration
	devices       *RegisteredDevices
}

// RegisteredDevices keeps track of all devices created by the forwarding plane
type RegisteredDevices struct {
	sync.Mutex
	devices map[string][]string
}

// CreateMetricsMonitor creates new metric monitoring instance
func CreateMetricsMonitor(requestPeriod time.Duration) *Metrics {
	return &Metrics{
		requestPeriod: requestPeriod,
		devices:       &RegisteredDevices{devices: map[string][]string{}},
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
	failedDevices := make(map[string][]string)
	devices.Lock()
	/* Loop through each registered device */
	for namespace, interfaces := range devices.devices {
		for _, device := range interfaces {
			metrics, err := getDeviceMetrics(device, namespace)
			if err != nil {
				logrus.Errorf("metrics: failed to extract metrics for device %s in namespace %s: %v", device, namespace, err)
				logrus.Errorf("metrics: removing device %s from device list", device)
				failedDevices[namespace] = append(failedDevices[namespace], device)
			} else {
				logrus.Infof("metrics: device %s, metrics - %v", device, metrics)
				stats[generateMetricsName(namespace, device)] = &crossconnect.Metrics{Metrics: metrics}
			}
		}
	}
	devices.Unlock()
	/* Update device list in case there are bad devices */
	if len(failedDevices) != 0 {
		for namespace, fails := range failedDevices {
			for _, fail := range fails {
				devices.UpdateDeviceList(map[string]string{namespace: fail}, false)
			}
		}
	}
	if len(stats) == 0 {
		return stats, fmt.Errorf("metrics: failed to extract metrics for any device in list: %v", devices.devices)
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
func (m *RegisteredDevices) UpdateDeviceList(devices map[string]string, connect bool) {
	found := false
	/* Add devices */
	m.Lock()
	defer m.Unlock()
	if connect {
		for namespace, device := range devices {
			devList, ok := m.devices[namespace]
			if !ok {
				/* Namespace is missing, so proceed adding the device */
				m.devices[namespace] = append(m.devices[namespace], device)
				continue
			}
			/* Namespace is present, search if there's such device in its list */
			for _, dev := range devList {
				if dev == device {
					/* Device requested for adding found */
					logrus.Errorf("metrics: device %s requested for add is already present in the devices list", device)
					found = true
					break
				}
			}
			/* Device requested for adding is not present, so we are free to add it */
			if !found {
				m.devices[namespace] = append(m.devices[namespace], device)
			}
		}
	} else {
		/* Delete devices */
		for namespace, device := range devices {
			/* Check if namespace is even present */
			devList, ok := m.devices[namespace]
			if !ok {
				logrus.Errorf("metrics: device %s with namespace %s requested for delete is already missing from the devices list", device, namespace)
				continue
			}
			/* Namespace is present, search if there's such device in its list */
			for i, dev := range devList {
				if dev == device {
					/* Device requested for deletion found */
					m.devices[namespace] = append(m.devices[namespace][:i], m.devices[namespace][i+1:]...)
					found = true
					break
				}
			}
			/* Device requested for deletion was not found */
			if !found {
				logrus.Errorf("metrics: device %s with namespace %s requested for delete is already missing from the devices list", device, namespace)
			}
			/* If there are no more devices associated with that namespace, delete it*/
			if len(m.devices[namespace]) == 0 {
				delete(m.devices, namespace)
			}
		}
	}
	logrus.Infof("metrics: device list - %v", m.devices)
}

// generateMetricsName generates a name for the metrics update
func generateMetricsName(namespace, device string) string {
	return device + "@" + namespace
}
