package pods

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NSCPodWebhook creates a new 'nsc' pod without init container
func NSCPodWebhook(name string, node *v1.Node) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"ns.networkservicemesh.io": "icmp-responder?app=icmp",
			},
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "alpine-img",
					Image:           "alpine:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command: []string{
						"tail", "-f", "/dev/null",
					},
				},
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

// WrongNSCPodWebhook creates a new 'nsc' pod with init container and nsm-annotation
func WrongNSCPodWebhook(name string, node *v1.Node) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"ns.networkservicemesh.io": "icmp-responder?app=icmp",
			},
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "alpine-img",
					Image:           "alpine:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command: []string{
						"tail", "-f", "/dev/null",
					},
				},
			},
			InitContainers: []v1.Container{
				{
					Name:            "nsm-init",
					Image:           "nsm-init:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
				},
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

// NSCPod creates a new 'nsc' pod with init container
func NSCPod(name string, node *v1.Node, env map[string]string) *v1.Pod {
	initContainer := newInitContainer(env)

	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
		},
		Spec: v1.PodSpec{
			ServiceAccountName: NSCServiceAccount,
			Containers: []v1.Container{
				newAlpineContainer(),
			},
			Volumes: []v1.Volume{
				spireVolume(),
			},
			InitContainers: []v1.Container{
				initContainer,
			},
			TerminationGracePeriodSeconds: &ZeroGraceTimeout,
		},
	}
	if node != nil {
		pod.Spec.NodeSelector = map[string]string{
			"kubernetes.io/hostname": node.Labels["kubernetes.io/hostname"],
		}
	}
	return pod
}

// NSCMonitorPod - creates client with nsm-init and nsm-monitor as side-car
func NSCMonitorPod(name string, node *v1.Node, env map[string]string) *v1.Pod {
	pod := NSCPod(name, node, env)
	pod.Spec.Containers = append(pod.Spec.Containers, newMonitorContainer(env))
	return pod
}

func newAlpineContainer() v1.Container {
	return v1.Container{
		Name:            "alpine-img",
		Image:           "alpine:latest",
		ImagePullPolicy: v1.PullIfNotPresent,
		Command: []string{
			"tail", "-f", "/dev/null",
		},
	}
}

func newInitContainer(env map[string]string) v1.Container {
	initContainer := containerMod(&v1.Container{
		Name:            "nsm-init",
		Image:           "networkservicemesh/nsm-init:latest",
		ImagePullPolicy: v1.PullIfNotPresent,
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
			},
		},
		VolumeMounts: []v1.VolumeMount{
			spireVolumeMount(),
		},
	})
	for k, v := range env {
		initContainer.Env = append(initContainer.Env,
			v1.EnvVar{
				Name:  k,
				Value: v,
			})
	}
	return initContainer
}
func newMonitorContainer(env map[string]string) v1.Container {
	result := containerMod(&v1.Container{
		Name:            "nsm-monitor",
		Image:           "networkservicemesh/nsm-monitor:latest",
		ImagePullPolicy: v1.PullIfNotPresent,
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
			},
		},
		VolumeMounts: []v1.VolumeMount{
			spireVolumeMount(),
		},
	})
	for k, v := range env {
		result.Env = append(result.Env,
			v1.EnvVar{
				Name:  k,
				Value: v,
			})
	}
	return result
}
