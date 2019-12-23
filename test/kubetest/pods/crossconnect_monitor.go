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

// CrossConnectMonitorDeployment creates a new 'crossconnect-monitor' deployment
func CrossConnectMonitorDeployment(namespace string, image string) *appsv1.Deployment {
	replicas := int32(1)
	ht := new(v1.HostPathType)
	*ht = v1.HostPathDirectoryOrCreate

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crossconnect-monitor",
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "crossconnect-monitor"},
			},
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "crossconnect-monitor",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "nsmgr-acc",
					Containers: []v1.Container{
						containerMod(&v1.Container{
							Name:            "crossconnect-monitor",
							Image:           image,
							ImagePullPolicy: v1.PullIfNotPresent,
							Env: []v1.EnvVar{
								{
									Name:  "INSECURE",
									Value: "false",
								},
								{
									Name:  "METRICS_COLLECTOR_ENABLED",
									Value: "true",
								},
								{
									Name:  "PROMETHEUS",
									Value: "true",
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "spire-agent-socket",
									MountPath: "/run/spire/sockets",
									ReadOnly:  true,
								},
							},
						}),
					},
					Volumes: []v1.Volume{
						{
							Name: "spire-agent-socket",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/run/spire/sockets",
									Type: ht,
								},
							},
						},
					},
				},
			},
		},
	}
}

// CrossConnectMonitorService creates a new 'crossconnect-monitor-svc' service
func CrossConnectMonitorService(namespace string) *v1.Service {
	targetPortIntVal := int32(9090)

	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "crossconnect-monitor-svc",
			Namespace: namespace,
			Labels: map[string]string{
				"app": "crossconnect-monitor",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Port: 9095, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{IntVal: targetPortIntVal}},
			},
			Selector: map[string]string{"app": "crossconnect-monitor"},
		},
	}
}
