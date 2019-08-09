package pods

import (
	"os"

	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"
)

const (
	NSMDHostSystemPath = "/go/src"
	NSMDHostRootEnv    = "NSMD_HOST_ROOT" // A host path for all sources.
)

// DefaultNSMD creates default variables for NSMD.
func DefaultNSMD() map[string]string {
	return map[string]string{
		nsmd.NsmdDeleteLocalRegistry: "true", // Do not use local registry restore for clients/NSEs
	}
}

func newNSMMount() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      "nsm-socket",
		MountPath: "/var/lib/networkservicemesh",
	}
}

func newNSMPluginMount() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      "nsm-plugin-socket",
		MountPath: "/var/lib/networkservicemesh/plugins",
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

type NSMgrContainerMode int8

const (
	NSMgrContainerNormal = 0
	NSMgrContainerRun    = 1
	NSMgrContainerDebug  = 2
)

type NSMgrPodConfig struct {
	Nsmd                NSMgrContainerMode // nsmd launch options - debug - for debug.sh, run - for run.sh
	NsmdK8s             NSMgrContainerMode // nsmd-k8s launch options - debug - for debug.sh, run - for run.sh
	NsmdP               NSMgrContainerMode // nsmdp launch options - debug - for debug.sh, run - for run.sh
	Variables           map[string]string
	DataplaneVariables  map[string]string
	liveness, readiness *v1.Probe
	Namespace           string
}

func NSMgrDevConfig(nsmd NSMgrContainerMode, nsmdp NSMgrContainerMode, nsmdk8s NSMgrContainerMode, namespace string) *NSMgrPodConfig {
	return &NSMgrPodConfig{
		Nsmd:      nsmd,
		NsmdK8s:   nsmdk8s,
		NsmdP:     nsmdp,
		Namespace: namespace,
	}
}

func NSMgrPod(name string, node *v1.Node, namespace string) *v1.Pod {
	return NSMgrPodWithConfig(name, node, &NSMgrPodConfig{
		Variables: DefaultNSMD(),
		Namespace: namespace,
	})
}
func NSMgrPodLiveCheck(name string, node *v1.Node, namespace string) *v1.Pod {
	return NSMgrPodWithConfig(name, node, &NSMgrPodConfig{
		liveness:  createProbe("/liveness"),
		readiness: createProbe("/readiness"),
		Variables: DefaultNSMD(),
		Namespace: namespace})
}

func NSMgrPodWithConfig(name string, node *v1.Node, config *NSMgrPodConfig) *v1.Pod {

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
				{
					Name: "nsm-plugin-socket",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Type: ht,
							Path: "/var/lib/networkservicemesh/plugins",
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
					Resources:       createDefaultResources(),
				}),
				containerMod(&v1.Container{
					Name:            "nsmd",
					Image:           "networkservicemesh/nsmd",
					ImagePullPolicy: v1.PullIfNotPresent,
					VolumeMounts:    []v1.VolumeMount{newNSMMount(), newNSMPluginMount()},
					LivenessProbe:   config.liveness,
					ReadinessProbe:  config.readiness,
					Resources:       createDefaultResources(),
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
						v1.EnvVar{
							Name:  namespace.NsmNamespaceEnv,
							Value: config.Namespace,
						},
					},
					Resources: createDefaultResources(),
				}),

			},
			TerminationGracePeriodSeconds: &ZeroGraceTimeout,
		},
	}
	if len(config.Variables) > 0 {
		for k, v := range config.Variables {
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
	if config.NsmdP != NSMgrContainerNormal {
		updateSpec(pod, 0, "nsmdp", config.NsmdP)
		updates++
	}
	if config.Nsmd != NSMgrContainerNormal {
		updateSpec(pod, 1, "nsmd", config.Nsmd)
		updates++
	}
	if config.NsmdK8s != NSMgrContainerNormal {
		updateSpec(pod, 2, "nsmd-k8s", config.NsmdK8s)
		updates++
	}

	if updates > 0 {
		pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
			Name: "src",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Type: ht,
					Path: getNSMDLocalHostSourcePath(),
				},
			},
		})
	}

	return pod
}

func getNSMDLocalHostSourcePath() string {
	root := os.Getenv(NSMDHostRootEnv)
	if root != "" {
		return root
	}
	return NSMDHostSystemPath
}

func updateSpec(pod *v1.Pod, index int, app string, mode NSMgrContainerMode) {
	ht := new(v1.HostPathType)
	*ht = v1.HostPathDirectoryOrCreate

	pod.Spec.Containers[index].VolumeMounts = append(pod.Spec.Containers[index].VolumeMounts, newDevSrcMount())
	pod.Spec.Containers[index].Command = []string{"bash"}
	if mode == NSMgrContainerDebug {
		pod.Spec.Containers[index].Args = []string{"/go/src/github.com/networkservicemesh/networkservicemesh/scripts/debug.sh", app}
	} else {
		pod.Spec.Containers[index].Args = []string{"/go/src/github.com/networkservicemesh/networkservicemesh/scripts/run.sh", app}
	}
	pod.Spec.Containers[index].Image = "networkservicemesh/devenv"
}
