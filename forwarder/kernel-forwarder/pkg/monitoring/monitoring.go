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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/metrics"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"
)

// Metrics monitoring instance type
type Metrics struct {
	requestPeriod time.Duration
	devices       *RegisteredDevices
}

// RegisteredDevices keeps track of all devices created by the forwarding plane
type RegisteredDevices struct {
	sync.Mutex
	devices map[string][]Device
}

// Device keeps information about a device
type Device struct {
	Name     string
	XconName string
}

// CreateMetricsMonitor creates new metric monitoring instance
func CreateMetricsMonitor(requestPeriod time.Duration) *Metrics {
	return &Metrics{
		requestPeriod: requestPeriod,
		devices:       &RegisteredDevices{devices: map[string][]Device{}},
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
		/* No need to process anything if list is empty */
		if len(devices.devices) != 0 {
			devices.Lock()
			/* Collect metrics for all present devices */
			stats, err := collectMetrics(devices)
			devices.Unlock()
			if err != nil {
				logrus.Warn("metrics: failed to collect metrics:", err)
			}
			/* Send metrics update */
			logrus.Debug("metrics: sending updates: ", stats)
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
	failedDevices := make(map[string][]Device)
	/* Loop through each registered device */
	for namespace, listOfDevices := range devices.devices {
		for _, device := range listOfDevices {
			deviceMetrics, err := getDeviceMetrics(device.Name, namespace)
			if err != nil {
				logrus.Warnf("metrics: failed to extract metrics for device %s in namespace %s: %v", device.Name, namespace, err)
				logrus.Warnf("metrics: removing device %s from device list", device.Name)
				failedDevices[namespace] = append(failedDevices[namespace], device)
			} else {
				logrus.Infof("metrics: device %s@%s, metrics - %v", device.Name, namespace, deviceMetrics)
				stats[generateMetricsName(device)] = &crossconnect.Metrics{Metrics: deviceMetrics}
			}
		}
	}
	/* Update device list in case there are bad devices */
	if len(failedDevices) != 0 {
		for namespace, fails := range failedDevices {
			for _, fail := range fails {
				devices.UpdateDeviceList(map[string]Device{namespace: fail}, false)
			}
		}
	}
	if len(stats) == 0 {
		return stats, errors.Errorf("metrics: failed to extract metrics for any device in list: %v", devices.devices)
	}
	return stats, nil
}

// getDeviceMetrics returns metrics for device in specific namespace
func getDeviceMetrics(device, nsInode string) (map[string]string, error) {
	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	/* Save current network namespace */
	hostNs, err := netns.Get()
	if err != nil {
		logrus.Errorf("metrics: failed getting host namespace: %v", err)
		return nil, err
	}
	logrus.Debug("metrics: host namespace: ", hostNs)
	defer func() {
		if err = hostNs.Close(); err != nil {
			logrus.Error("metrics: failed closing host namespace handle: ", err)
		}
		logrus.Debug("metrics: closed host namespace handle: ", hostNs)
	}()

	/* Get namespace handle - destination */
	dstHandle, err := fs.GetNsHandleFromInode(nsInode)
	if err != nil {
		logrus.Errorf("metrics: failed to get destination namespace handle - %v", err)
		return nil, err
	}
	/* If successful, don't forget to close the handler upon exit */
	defer func() {
		if err = dstHandle.Close(); err != nil {
			logrus.Error("metrics: error when closing destination handle: ", err)
		}
		logrus.Debug("metrics: closed destination handle: ", dstHandle, nsInode)
	}()
	logrus.Debug("metrics: opened destination handle: ", dstHandle, nsInode)

	/* Switch to the new namespace */
	if err = netns.Set(dstHandle); err != nil {
		logrus.Errorf("metrics: failed switching to destination namespace: %v", err)
		return nil, err
	}
	logrus.Debug("metrics: switched to destination namespace: ", dstHandle)

	/* Don't forget to switch back to the host namespace */
	defer func() {
		if err = netns.Set(hostNs); err != nil {
			logrus.Errorf("metrics: failed switching back to host namespace: %v", err)
		}
		logrus.Debug("metrics: switched back to host namespace: ", hostNs)
	}()

	/* Get a link for the interface name */
	link, err := netlink.LinkByName(device)
	if err != nil {
		logrus.Errorf("metrics: failed to lookup %q, %v", device, err)
		return nil, err
	}
	/* 6. Save statistics in metrics map */
	metricsMap := make(map[string]string)
	metricsMap["rx_bytes"] = fmt.Sprint(link.Attrs().Statistics.RxBytes)
	metricsMap["tx_bytes"] = fmt.Sprint(link.Attrs().Statistics.TxBytes)
	metricsMap["rx_packets"] = fmt.Sprint(link.Attrs().Statistics.RxPackets)
	metricsMap["tx_packets"] = fmt.Sprint(link.Attrs().Statistics.TxPackets)
	metricsMap["rx_error_packets"] = fmt.Sprint(link.Attrs().Statistics.RxErrors)
	metricsMap["tx_error_packets"] = fmt.Sprint(link.Attrs().Statistics.TxErrors)

	return metricsMap, nil
}

// UpdateDeviceList keeps track of the devices being handled by the Kernel forwarding plane
func (m *RegisteredDevices) UpdateDeviceList(devices map[string]Device, connect bool) {
	found := false
	/* Loop through each updated device - either added or deleted */
	for namespace, device := range devices {
		devList, ok := m.devices[namespace]
		if !ok {
			if connect {
				/* Add: Namespace is missing, so we are free to add the device */
				m.devices[namespace] = append(m.devices[namespace], device)
			} else {
				/* Delete: Namespace is missing, so there's no point to look for a device associated with it */
				logrus.Errorf("metrics: device %s with namespace %s requested for delete is already missing", device, namespace)
			}
			continue
		}
		/* Namespace is present, search if the device is found in its list */
		for i, dev := range devList {
			if dev.Name == device.Name {
				if connect {
					/* Add: The device we want to add is already there */
					logrus.Errorf("metrics: device %s requested for add is already present", device.Name)
				} else {
					/* Delete: Found the device we want to delete */
					m.devices[namespace] = append(m.devices[namespace][:i], m.devices[namespace][i+1:]...)
				}
				found = true
				break
			}
		}
		/* There's such a namespace, but the requested device is not found in its list */
		if !found {
			if connect {
				/* Add: We are free to add it */
				m.devices[namespace] = append(m.devices[namespace], device)
			} else {
				/* Delete: There's really no such device found for deletion */
				logrus.Errorf("metrics: device %s with namespace %s requested for delete is already missing", device.Name, namespace)
			}
		}
		/* If there are no more devices associated with that namespace, delete it */
		if !connect && len(m.devices[namespace]) == 0 {
			delete(m.devices, namespace)
		}
	}
	logrus.Infof("metrics: device list - %v", m.devices)
}

// generateMetricsName generates a name for the metrics update
func generateMetricsName(device Device) string {
	return device.XconName
}
