package pods

import (
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var DefaultNSMD = map[string]string{
}

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

func newDevSrcMount() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      "src",
		MountPath: "/go/src",
	}
}

type NSMDPodMode int8

const (
	NSMDPodNormal = 0
	NSMDPodRun = 1
	NSMDPodDebug = 2
)

type NSMDPodConfig struct {
	Nsmd                NSMDPodMode // nsmd launch options - debug - for debug.sh, run - for run.sh
	NsmdK8s             NSMDPodMode // nsmd-k8s launch options - debug - for debug.sh, run - for run.sh
	NsmdP               NSMDPodMode // nsmdp launch options - debug - for debug.sh, run - for run.sh
	Variables           map[string]string
	liveness, readiness *v1.Probe
}

func NSMDDevConfig(nsmd NSMDPodMode, nsmdp NSMDPodMode, nsmdk8s NSMDPodMode) *NSMDPodConfig {
	return &NSMDPodConfig{
		Nsmd:         nsmd,
		NsmdK8s:      nsmdk8s,
		NsmdP:        nsmdp,
	}
}


func NSMDPod(name string, node *v1.Node) *v1.Pod {
	return NSMDPodWithConfig(name, node, &NSMDPodConfig{
		Variables: DefaultNSMD,
	})
}
func NSMDPodLiveCheck(name string, node *v1.Node) *v1.Pod {
	return NSMDPodWithConfig(name, node, &NSMDPodConfig{
		liveness:  createProbe("/liveness"),
		readiness: createProbe("/readiness")})
}

func NSMDPodWithConfig(name string, node *v1.Node, config *NSMDPodConfig) *v1.Pod {
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
					LivenessProbe:   config.liveness,
					ReadinessProbe:  config.readiness,
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
	if len(config.Variables) > 0 {
		for k, v := range (config.Variables) {
			pod.Spec.Containers[1].Env = append(pod.Spec.Containers[1].Env, v1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}
	if node != nil {
		pod.Spec.NodeSelector = map[string]string{
			"kubernetes.io/hostname": node.Labels["kubernetes.io/hostname"],
		}
	}

	updates := 0
	if config.NsmdP != NSMDPodNormal {
		updateSpec(pod, 0, "nsmdp", config.NsmdP)
		updates++
	}
	if config.Nsmd != NSMDPodNormal {
		updateSpec(pod, 1, "nsmd", config.Nsmd)
		updates++
	}
	if config.NsmdK8s != NSMDPodNormal {
		updateSpec(pod, 2, "nsmd-k8s", config.NsmdK8s)
		updates++
	}

	if updates > 0 {
		pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
			Name: "src",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Type: ht,
					Path: "/go/src",
				},
			},
		}, )
	}

	return pod
}

func updateSpec(pod *v1.Pod, index int, app string, mode NSMDPodMode ) {
	ht := new(v1.HostPathType)
	*ht = v1.HostPathDirectoryOrCreate

	pod.Spec.Containers[index].VolumeMounts = append(pod.Spec.Containers[index].VolumeMounts, newDevSrcMount())
	pod.Spec.Containers[index].Command = []string{"bash"}
	if mode == NSMDPodDebug {
		pod.Spec.Containers[index].Args = []string{"/go/src/github.com/networkservicemesh/networkservicemesh/scripts/debug.sh", app}
	} else {
		pod.Spec.Containers[index].Args = []string{"/go/src/github.com/networkservicemesh/networkservicemesh/scripts/run.sh", app}
	}
	pod.Spec.Containers[index].Image = "networkservicemesh/devenv"
}