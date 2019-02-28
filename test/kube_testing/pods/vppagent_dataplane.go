package pods

import (
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PostMortemDataLocation    = "/var/tmp/nsm-postmortem"
	VppPostmortemDataLocation = "/var/tmp/nsm-postmortem/vpp-dataplane"
	VppBacktracePattern       = "*.vpp_backtrace"
	BacktracePattern          = "*backtrace"
)

func VPPDataplanePod(name string, node *v1.Node) *v1.Pod {
	return createVPPDataplanePod(name, node, nil, nil)
}

func VPPDataplanePodLiveCheck(name string, node *v1.Node) *v1.Pod {
	return createVPPDataplanePod(name, node, createProbe("/liveness"), createProbe("/readiness"))
}

func createVPPDataplanePod(name string, node *v1.Node, liveness, readiness *v1.Probe) *v1.Pod {
	ht := new(v1.HostPathType)
	*ht = v1.HostPathDirectoryOrCreate

	priv := true

	mode := v1.MountPropagationBidirectional
	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
		},
		Spec: v1.PodSpec{
			HostPID: true,
			Volumes: []v1.Volume{
				{
					Name: "workspace",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Type: ht,
							Path: "/var/lib/networkservicemesh",
						},
					},
				},
			},
			Containers: []v1.Container{
				containerMod(&v1.Container{
					Name:            "vppagent-dataplane",
					Image:           "networkservicemesh/vppagent-dataplane-dev:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					VolumeMounts: []v1.VolumeMount{
						v1.VolumeMount{
							Name:             "workspace",
							MountPath:        "/var/lib/networkservicemesh/",
							MountPropagation: &mode,
						},
					},
					Env: []v1.EnvVar{
						v1.EnvVar{
							Name: "NSM_DATAPLANE_SRC_IP",
							ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{
									FieldPath: "status.podIP",
								},
							},
						},
						v1.EnvVar{
							Name:  "EXIT_ON_CRASH",
							Value: "false",
						},
					},
					SecurityContext: &v1.SecurityContext{
						Privileged: &priv,
					},
					LivenessProbe:  liveness,
					ReadinessProbe: readiness,
				}),
			},
		},
	}
	if node != nil {
		pod.Spec.NodeSelector = map[string]string{
			"kubernetes.io/hostname": node.Labels["kubernetes.io/hostname"],
		}
	}
	return pod
}
