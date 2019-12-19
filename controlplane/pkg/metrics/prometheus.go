// Copyright (c) 2019 VMware, Inc.
//
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

// Package metrics provides methods tracking metrics in Prometheus
package metrics

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
)

const (
	prometheusPort = "0.0.0.0:9090"
	// PrometheusEnv is a boolean env var to
	// enable/disable Prometheus. Values true or false
	PrometheusEnv = "PROMETHEUS"
	// PrometheusDefault is the default value for PrometheusEnv
	PrometheusDefault = false
)

// PrometheusMetric contains vector name
// and prometheus.GaugeVec reference
type PrometheusMetric struct {
	Name  string
	Vec   *prometheus.GaugeVec
	Value string
}

const (
	// RxBytes is vector name for "rx_bytes"
	RxBytes = "rx_bytes"
	// TxBytes is vector name for "tx_bytes"
	TxBytes = "tx_bytes"
	// RxPackets is vector name for "rx_packets"
	RxPackets = "rx_packets"
	// TxPackets is vector name for "tx_packets"
	TxPackets = "tx_packets"
	// RxErrorPackets is vector name for "rx_error_packets"
	RxErrorPackets = "rx_error_packets"
	// TxErrorPackets is vector name for "tx_error_packets"
	TxErrorPackets = "tx_error_packets"

	// SrcPodKey is vector label for source pod
	SrcPodKey = "src_pod"
	// SrcNamespaceKey is vector label for source pod namespace
	SrcNamespaceKey = "src_namespace"
	// DstPodKey is vector label for dest pod
	DstPodKey = "dst_pod"
	// DstNamespaceKey is vector label for dest pod namespace
	DstNamespaceKey = "dst_namespace"
)

// PrometheusMetricsContext is metrics context,
// containing source and destination pods (and their namespaces)
type PrometheusMetricsContext struct {
	// SrcPodName is the name of the source pod in a connection
	SrcPodName string
	// DstPodName is the name of the dest pod in a connection
	DstPodName string
	// SrcNamespace is the namespace of the source pod in a connection
	SrcNamespace string
	// DstNamespace is the namespace of the dest pod in a connection
	DstNamespace string
	// Metrics is the metrics data for the connection
	Metrics map[string]string
}

// BuildPrometheusMetricsContext builds single metrics context,
// i.e. source and destination pods (and their namespaces)
// whose connection the metrics are going to describe
func BuildPrometheusMetricsContext(srcPodName, srcNamespace, dstPodName, dstNamespace string) PrometheusMetricsContext {
	ctx := PrometheusMetricsContext{
		SrcPodName:   srcPodName,
		SrcNamespace: srcNamespace,
		DstPodName:   dstPodName,
		DstNamespace: dstNamespace,
	}

	return ctx
}

// BuildPrometheusMetrics builds prometheus
// Gauge vectiors for rx_bytes, tx_bytes, rx_packets,
// tx_packets, rx_error_packets and tx_error_packets
func BuildPrometheusMetrics() []PrometheusMetric {
	prometheusMetrics := make([]PrometheusMetric, 0, 6)

	rxBytes := buildPrometheusMetric(RxBytes)
	prometheusMetrics = append(prometheusMetrics, rxBytes)

	txBytes := buildPrometheusMetric(TxBytes)
	prometheusMetrics = append(prometheusMetrics, txBytes)

	rxPackets := buildPrometheusMetric(RxPackets)
	prometheusMetrics = append(prometheusMetrics, rxPackets)

	txPackets := buildPrometheusMetric(TxPackets)
	prometheusMetrics = append(prometheusMetrics, txPackets)

	rxErrorPackets := buildPrometheusMetric(RxErrorPackets)
	prometheusMetrics = append(prometheusMetrics, rxErrorPackets)

	txErrorPackets := buildPrometheusMetric(TxErrorPackets)
	prometheusMetrics = append(prometheusMetrics, txErrorPackets)

	return prometheusMetrics
}

func buildPrometheusMetric(metricType string) PrometheusMetric {
	metricGaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metricType,
			Help: fmt.Sprintf("%s transmissed", metricType),
		},
		[]string{SrcPodKey, SrcNamespaceKey, DstPodKey, DstNamespaceKey},
	)

	if err := prometheus.Register(metricGaugeVec); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			metricGaugeVec = are.ExistingCollector.(*prometheus.GaugeVec)
			logrus.Infof("using already registered vector %v", metricGaugeVec)
		} else {
			logrus.Infof("failed to register vector %v, err: %v", metricGaugeVec, err)
		}
	} else {
		logrus.Infof("successfully registered vector %v", metricGaugeVec)
	}

	res := PrometheusMetric{
		Name: metricType,
		Vec:  metricGaugeVec,
	}

	return res
}

// CollectAllMetrics collects metrcis from the crossconnect.Metrics data,
// and based on the given context (i.e. src and destination connection
// the metrics are for) tracks that data in the corresponding vectors
func CollectAllMetrics(ctx PrometheusMetricsContext, prometheusMetrics []PrometheusMetric, metrics *crossconnect.Metrics) {
	for i := range prometheusMetrics {
		if metrics != nil {
			if _, ok := metrics.Metrics[prometheusMetrics[i].Name]; ok {
				prometheusMetrics[i].collect(ctx, metrics.Metrics[prometheusMetrics[i].Name])
			}
		}
	}
}

func (pm *PrometheusMetric) collect(ctx PrometheusMetricsContext, trafficSize string) {
	flmetric, err := strconv.ParseFloat(trafficSize, 64)
	if err != nil {
		logrus.Infof("failed to convert trafficSize %s to float64", trafficSize)
	}
	labels := map[string]string{
		SrcPodKey:       ctx.SrcPodName,
		SrcNamespaceKey: ctx.SrcNamespace,
		DstPodKey:       ctx.DstPodName,
		DstNamespaceKey: ctx.DstNamespace,
	}
	pm.Vec.With(labels).Set(flmetric)
}

// RunPrometheusMetricsServer is running Prometheus to handle traffic metrics
// Stop is deferred in the client implementation
func RunPrometheusMetricsServer() {
	logrus.Infof("Starting Prometheus server")
	server := GetPrometheusMetricsServer()
	err := server.ListenAndServe()
	if err != nil {
		logrus.Infof("failed to listen and serve cross-connect metrics")
	}
}

// GetPrometheusMetricsServer creates a http server
// to expose the metrics to. Prometheus is going
// to collect the data from that server
func GetPrometheusMetricsServer(prometheusAddress ...string) *http.Server {
	prometheusHostPort := ""
	if len(prometheusAddress) > 0 {
		prometheusHostPort = prometheusAddress[0]
	} else {
		prometheusHostPort = prometheusPort
	}
	server := &http.Server{Addr: prometheusHostPort}
	http.Handle("/metrics", promhttp.Handler())
	return server
}
