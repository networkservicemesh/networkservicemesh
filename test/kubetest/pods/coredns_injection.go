package pods

import (
	v1 "k8s.io/api/core/v1"
)

func InjectCorednsWithSharedFolder(pod *v1.Pod) {
	pod.Spec.Containers = append(pod.Spec.Containers,
		v1.Container{
			Name:            "coredns",
			Image:           "coredns/coredns:latest",
			ImagePullPolicy: v1.PullIfNotPresent,
			Args:            []string{"-conf", "/etc/coredns/Corefile"},
		})
	pod.Spec.Containers[len(pod.Spec.Containers)-1].VolumeMounts = []v1.VolumeMount{{
		ReadOnly:  false,
		Name:      "empty-dir-volume",
		MountPath: "/etc/coredns",
	}}
	pod.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{{
		ReadOnly:  false,
		Name:      "empty-dir-volume",
		MountPath: "/etc/coredns",
	}}
	pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
		Name: "empty-dir-volume",
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{
				Medium:    v1.StorageMediumDefault,
				SizeLimit: nil,
			},
		},
	})
	setupDNSConfig(pod)
}

func InjectCoredns(pod *v1.Pod, corednsConfigName string) {
	pod.Spec.Containers = append(pod.Spec.Containers,
		v1.Container{
			Name:            "coredns",
			Image:           "coredns/coredns:latest",
			ImagePullPolicy: v1.PullIfNotPresent,
			Args:            []string{"-conf", "/etc/coredns/Corefile"},
			VolumeMounts: []v1.VolumeMount{{
				ReadOnly:  false,
				Name:      "config-volume",
				MountPath: "/etc/coredns",
			}},
		})

	pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
		Name: "config-volume",
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: corednsConfigName},
				Items: []v1.KeyToPath{{
					Key:  "Corefile",
					Path: "Corefile",
				}},
			},
		},
	})

	setupDNSConfig(pod)
}

func setupDNSConfig(pod *v1.Pod) {
	pod.Spec.DNSPolicy = v1.DNSNone
	pod.Spec.DNSConfig = &v1.PodDNSConfig{}
	pod.Spec.DNSConfig.Nameservers = []string{"127.0.0.1", "10.96.0.10"}
	pod.Spec.DNSConfig.Searches = []string{"default.svc.cluster.local", "svc.cluster.local", "cluster.local"}
}
