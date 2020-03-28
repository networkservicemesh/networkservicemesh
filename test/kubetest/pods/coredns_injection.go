package pods

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

//InjectCorednsWithSharedFolder - Injects coredns container and configure the DnsConfig for template.
//Also makes shared folder between coredns container and first container of template
func InjectCorednsWithSharedFolder(template *v1.Pod) {
	template.Spec.Containers = append(template.Spec.Containers,
		v1.Container{
			Name:            "coredns",
			Image:           "networkservicemesh/coredns:master",
			ImagePullPolicy: v1.PullIfNotPresent,
			Args:            []string{"-conf", "/etc/coredns/Corefile"},
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
				},
			},
			VolumeMounts: []v1.VolumeMount{{
				ReadOnly:  false,
				Name:      "empty-dir-volume",
				MountPath: "/etc/coredns",
			}},
		})
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
	template.Spec.InitContainers = append([]v1.Container{newDNSInitContainer(nil)}, template.Spec.InitContainers...)
}

//InjectCoredns - Injects coredns container and configure the DnsConfig for template.
func InjectCoredns(pod *v1.Pod, corednsConfigName string) *v1.Pod {
	pod.Spec.Containers = append(pod.Spec.Containers,
		v1.Container{
			Name:            "coredns",
			Image:           "networkservicemesh/coredns:master",
			ImagePullPolicy: v1.PullIfNotPresent,
			Args:            []string{"-conf", "/etc/coredns/Corefile"},
			VolumeMounts: []v1.VolumeMount{{
				ReadOnly:  false,
				Name:      "config-volume",
				MountPath: "/etc/coredns",
			}},
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
				},
			},
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
	return pod
}
