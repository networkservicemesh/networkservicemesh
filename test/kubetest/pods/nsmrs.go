package pods

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultNSMRS creates default variables for NSMRS.
func DefaultNSMRS() map[string]string {
	return map[string]string{}
}

const ()

// NSMRSPodConfig - configuration required for NSMRS Pod creating (environment variables)
type NSMRSPodConfig struct {
	Variables map[string]string
}

// NSMRSPod - create NSMRS pod with default config
func NSMRSPod(name string, node *v1.Node) *v1.Pod {
	return NSMRSPodWithConfig(name, node, &NSMgrPodConfig{
		Variables: DefaultNSMRS(),
	})
}

// NSMRSPodWithConfig - create NSMRS pod with custom config
func NSMRSPodWithConfig(name string, node *v1.Node, config *NSMgrPodConfig) *v1.Pod {

	ht := new(v1.HostPathType)
	*ht = v1.HostPathDirectoryOrCreate

	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				containerMod(&v1.Container{
					Name:            "nsmrs",
					Image:           "networkservicemesh/nsmrs",
					ImagePullPolicy: v1.PullIfNotPresent,
					Resources:       createDefaultResources(),
					Ports: []v1.ContainerPort{
						{
							HostPort:      80,
							ContainerPort: 5006,
						},
					},
				}),
			},
		},
	}
	if len(config.Variables) > 0 {
		for k, v := range config.Variables {
			for i := range pod.Spec.Containers {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, v1.EnvVar{
					Name:  k,
					Value: v,
				})
			}
		}
	}
	if node != nil {
		pod.Spec.NodeSelector = map[string]string{
			"kubernetes.io/hostname": node.Labels["kubernetes.io/hostname"],
		}
	}

	return pod
}
