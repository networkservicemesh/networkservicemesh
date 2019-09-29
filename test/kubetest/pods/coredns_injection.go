package pods

import (
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/k8s/cmd/nsm-coredns/env"
)

//InjectNSMCorednsWithSharedFolder - Injects nsm-coredns container and configure the DnsConfig for template.
//Also makes shared folder between nsm-coredns container and first container of template
func InjectNSMCorednsWithSharedFolder(template *v1.Pod) {
	template.Spec.Containers = append(template.Spec.Containers,
		containerMod(&v1.Container{
			Name:            "nsm-coredns",
			Image:           "networkservicemesh/nsm-coredns:latest",
			ImagePullPolicy: v1.PullIfNotPresent,
			Args:            []string{"-conf", "/etc/coredns/Corefile"},
			Env: []v1.EnvVar{
				{
					Name:  env.UseUpdateAPIEnv.Name(),
					Value: "true",
				},
				{
					Name:  env.UpdateAPIClientSock.Name(),
					Value: "/etc/coredns/client.sock",
				},
			},
		}))
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
}

//InjectNSMCoredns - Injects nsm-coredns container and configure the DnsConfig for template.
func InjectNSMCoredns(pod *v1.Pod, corednsConfigName string) *v1.Pod {
	pod.Spec.Containers = append(pod.Spec.Containers,
		containerMod(&v1.Container{
			Name:            "nsm-coredns",
			Image:           "networkservicemesh/nsm-coredns:latest",
			ImagePullPolicy: v1.PullIfNotPresent,
			Args:            []string{"-conf", "/etc/coredns/Corefile"},
			VolumeMounts: []v1.VolumeMount{{
				ReadOnly:  false,
				Name:      "config-volume",
				MountPath: "/etc/coredns",
			}},
		}))

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
	return pod
}
