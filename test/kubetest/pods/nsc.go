package pods

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NSCPodDns(name string, node *v1.Node, corednsConfigName string) * v1.Pod {
	config := map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   "icmp-responder",
	}
	result := NSCPod(name, node, config)
	result.Spec.Containers = append(result.Spec.Containers,
	v1.Container{
		Name:"coredns",
		Image:"coredns/coredns:1.5.0",
		ImagePullPolicy: v1.PullIfNotPresent,
		Args: []string{"-conf", "/etc/coredns/Corefile"},
		VolumeMounts: []v1.VolumeMount{{
			ReadOnly:true,
			Name:"config-volume",
			MountPath:"/etc/coredns",
		},

		},

	})
	result.Spec.DNSPolicy = v1.DNSNone
	result.Spec.DNSConfig = &v1.PodDNSConfig{}
	result.Spec.DNSConfig.Nameservers = []string{"127.0.0.1", "10.96.0.10"}
	result.Spec.DNSConfig.Searches = []string{"default.svc.cluster.local","svc.cluster.local", "cluster.local"}
	result.Spec.Volumes = append(result.Spec.Volumes, v1.Volume{
		Name: "config-volume",
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name:corednsConfigName},
				Items: []v1.KeyToPath{{
					Key:"Corefile",
					Path:"Corefile",
				},
				},
			},
		},
	})

	return result
}

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

// NSCPod creates a new 'nsc' pod with init container
func NSCPod(name string, node *v1.Node, env map[string]string) *v1.Pod {
	initContainer := containerMod(&v1.Container{
		Name:            "nsm-init",
		Image:           "networkservicemesh/nsm-init:latest",
		ImagePullPolicy: v1.PullIfNotPresent,
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
			},
		},
	})
	for k, v := range env {
		initContainer.Env = append(initContainer.Env,
			v1.EnvVar{
				Name:  k,
				Value: v,
			})
	}

	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
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
