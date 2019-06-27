package pods

import (
	v1 "k8s.io/api/core/v1"
)

//InjectCorednsWithSharedFolder - Injects coredns container and configure the DnsConfig for template.
//Also makes shared folder between coredns container and first container of template
func InjectCorednsWithSharedFolder(template *v1.Pod) {
	template.Spec.Containers = append(template.Spec.Containers,
		v1.Container{
			Name:            "coredns",
			Image:           "coredns/coredns:latest",
			ImagePullPolicy: v1.PullIfNotPresent,
			Args:            []string{"-conf", "/etc/coredns/Corefile"},
		})
	template.Spec.Containers[len(template.Spec.Containers)-1].VolumeMounts = []v1.VolumeMount{{
		ReadOnly:  false,
		Name:      "empty-dir-volume",
		MountPath: "/etc/coredns",
	}}
	template.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{{
		ReadOnly:  false,
		Name:      "empty-dir-volume",
		MountPath: "/etc/coredns",
	}}
	template.Spec.Volumes = append(template.Spec.Volumes, v1.Volume{
		Name: "empty-dir-volume",
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{
				Medium:    v1.StorageMediumDefault,
				SizeLimit: nil,
			},
		},
	})
	setupDNSConfig(template)
}

//InjectCoredns - Injects coredns container and configure the DnsConfig for template.
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
