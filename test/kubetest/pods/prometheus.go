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

//nolint
package pods

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// PrometheusConfigMap creates a new 'prometheus-server' config map
func PrometheusConfigMap(namespace string) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-server",
			Namespace: namespace,
			Labels: map[string]string{
				"name": "prometheus-server",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "ConfigMap",
		},
		Data: map[string]string{
			"prometheus.yml": "global:\n" +
				"  \"scrape_interval\": \"5s\"\n" +
				"  \"evaluation_interval\": \"5s\"\n\n" +
				"scrape_configs:\n" +
				"- job_name: 'prometheus'\n" +
				"  scrape_interval: 5s\n" +
				"  static_configs:\n" +
				"    - targets: ['localhost:9090']\n" +
				"- job_name: 'crossconnect-monitor'\n" +
				"  static_configs:\n" +
				"    - targets: ['crossconnect-monitor-svc:9095']\n" +
				"- job_name: 'kubernetes-apiservers'\n" +
				"  kubernetes_sd_configs:\n" +
				"  - role: endpoints\n" +
				"  scheme: https\n" +
				"  tls_config:\n" +
				"    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt\n" +
				"  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token\n" +
				"  relabel_configs:\n" +
				"  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]\n" +
				"    action: keep\n" +
				"    regex: default;kubernetes;https\n\n" +
				"- job_name: 'kubernetes-nodes'\n" +
				"  scheme: https\n" +
				"  tls_config:\n" +
				"    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt\n" +
				"  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token\n" +
				"  kubernetes_sd_configs:\n" +
				"  - role: node\n" +
				"  relabel_configs:\n" +
				"  - action: labelmap\n" +
				"    regex: __meta_kubernetes_node_label_(.+)\n" +
				"  - target_label: __address__\n" +
				"    replacement: kubernetes.default.svc:443\n" +
				"  - source_labels: [__meta_kubernetes_node_name]\n" +
				"    regex: (.+)\n" +
				"    target_label: __metrics_path__\n" +
				"    replacement: /api/v1/nodes/${1}/proxy/metrics\n\n" +
				"- job_name: 'kubernetes-pods'\n" +
				"  kubernetes_sd_configs:\n" +
				"  - role: pod\n" +
				"  relabel_configs:\n" +
				"  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]\n" +
				"    action: keep\n" +
				"    regex: true\n" +
				"  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]\n" +
				"    action: replace\n" +
				"    target_label: __metrics_path__\n" +
				"    regex: (.+)\n" +
				"  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]\n" +
				"    action: replace\n" +
				"    regex: ([^:]+)(?::\\d+)?;(\\d+)\n" +
				"    replacement: $1:$2\n" +
				"    target_label: __address__\n" +
				"  - action: labelmap\n" +
				"    regex: __meta_kubernetes_pod_label_(.+)\n" +
				"  - source_labels: [__meta_kubernetes_namespace]\n" +
				"    action: replace\n" +
				"    target_label: kubernetes_namespace\n" +
				"  - source_labels: [__meta_kubernetes_pod_name]\n" +
				"    action: replace\n" +
				"    target_label: kubernetes_pod_name\n\n" +
				"- job_name: 'kubernetes-cadvisor'\n" +
				"  scheme: https\n" +
				"  tls_config:\n" +
				"    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt\n" +
				"  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token\n" +
				"  kubernetes_sd_configs:\n" +
				"  - role: node\n" +
				"  relabel_configs:\n" +
				"  - action: labelmap\n" +
				"    regex: __meta_kubernetes_node_label_(.+)\n" +
				"  - target_label: __address__\n" +
				"    replacement: kubernetes.default.svc:443\n" +
				"  - source_labels: [__meta_kubernetes_node_name]\n" +
				"    regex: (.+)\n" +
				"    target_label: __metrics_path__\n" +
				"    replacement: /api/v1/nodes/${1}/proxy/metrics/cadvisor\n\n" +
				"- job_name: 'kubernetes-service-endpoints'\n" +
				"  kubernetes_sd_configs:\n" +
				"  - role: endpoints\n" +
				"  relabel_configs:\n" +
				"  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]\n" +
				"    action: keep\n" +
				"    regex: true\n" +
				"  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scheme]\n" +
				"    action: replace\n" +
				"    target_label: __scheme__\n" +
				"    regex: (https?)\n" +
				"  - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]\n" +
				"    action: replace\n" +
				"    target_label: __metrics_path__\n" +
				"    regex: (.+)\n" +
				"  - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]\n" +
				"    action: replace\n" +
				"    target_label: __address__\n" +
				"    regex: ([^:]+)(?::\\d+)?;(\\d+)\n" +
				"    replacement: $1:$2\n" +
				"  - action: labelmap\n" +
				"    regex: __meta_kubernetes_service_label_(.+)\n" +
				"  - source_labels: [__meta_kubernetes_namespace]\n" +
				"    action: replace\n" +
				"    target_label: kubernetes_namespace\n" +
				"  - source_labels: [__meta_kubernetes_service_name]\n" +
				"    action: replace\n" +
				"    target_label: kubernetes_name\n",
		},
	}
}

// PrometheusDeployment creates a new 'prometheus-server' deployment
func PrometheusDeployment(namespace string) *appsv1.Deployment {
	replicas := int32(1)
	defaultMode := int32(420)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-server",
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "prometheus-server"},
			},
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "prometheus-server",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						containerMod(&v1.Container{
							Name:            "prometheus",
							Image:           "prom/prometheus:v2.2.1",
							ImagePullPolicy: v1.PullIfNotPresent,
							Args: []string{
								"--config.file=/etc/prometheus/prometheus.yml",
								"--storage.tsdb.path=/prometheus/",
							},
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 9090,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "prometheus-config-volume",
									MountPath: "/etc/prometheus/",
								},
								{
									Name:      "prometheus-storage-volume",
									MountPath: "/prometheus/",
								},
							},
						}),
					},
					Volumes: []v1.Volume{
						{
							Name: "prometheus-config-volume",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									DefaultMode:          &defaultMode,
									LocalObjectReference: v1.LocalObjectReference{Name: "prometheus-server"},
								},
							},
						},
						{
							Name: "prometheus-storage-volume",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}

// PrometheusService creates a new 'prometheus-server' service
func PrometheusService(namespace string) *v1.Service {
	targetPortIntVal := int32(9090)
	serviceType := v1.ServiceType("ClusterIP")

	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-server",
			Namespace: namespace,
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "9090",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Port: 80, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{IntVal: targetPortIntVal}},
			},
			Selector: map[string]string{"app": "prometheus-server"},
			Type:     serviceType,
		},
	}
}
