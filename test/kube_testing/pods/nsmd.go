package pods

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newNSMMount() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      "nsm-socket",
		MountPath: "/var/lib/networkservicemesh",
	}
}

func newDevMount() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      "kubelet-socket",
		MountPath: "/var/lib/kubelet/device-plugins",
	}
}

func NSMDPod(name string, node *v1.Node) *v1.Pod {
	ht := new(v1.HostPathType)
	*ht = v1.HostPathDirectoryOrCreate

	nodeName := "master"
	if node != nil {
		nodeName = node.Name
	}

	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
			//Kind: "DaemonSet",
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{
				{
					Name: "kubelet-socket",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Type: ht,
							Path: "/var/lib/kubelet/device-plugins",
						},
					},
				},
				{
					Name: "nsm-socket",
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
					Name:            "nsmdp",
					Image:           "networkservicemesh/nsmdp",
					ImagePullPolicy: v1.PullIfNotPresent,
					VolumeMounts:    []v1.VolumeMount{newDevMount(), newNSMMount()},
				}),
				containerMod(&v1.Container{
					Name:            "nsmd",
					Image:           "networkservicemesh/nsmd",
					ImagePullPolicy: v1.PullIfNotPresent,
					VolumeMounts:    []v1.VolumeMount{newNSMMount()},
				}),
				containerMod(&v1.Container{
					Name:            "nsmd-k8s",
					Image:           "networkservicemesh/nsmd-k8s",
					ImagePullPolicy: v1.PullIfNotPresent,
					Env: []v1.EnvVar{
						v1.EnvVar{
							Name:  "NODE_NAME",
							Value: nodeName,
						},
					},
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
