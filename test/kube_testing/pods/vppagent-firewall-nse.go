package pods

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func VppAgentFirewallNSEConfigMapIcmpHttp(name string, namespace string) *v1.ConfigMap {

	return &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			Name:      name + "-config-file",
			Namespace: namespace,
		},
		TypeMeta: v12.TypeMeta{
			Kind: "ConfigMap",
		},
		Data: map[string]string{
			"config.yaml": "aclRules:\n" +
				"  \"Allow ICMP\": \"action=reflect,icmptype=8\"\n" +
				"  \"Allow TCP 80\": \"action=reflect,tcplowport=80,tcpupport=80\"\n",
		},
	}
}

func VppAgentFirewallNSEPodWithConfigMap(name string, node *v1.Node, env map[string]string) *v1.Pod {
	p := VppAgentFirewallNSEPod(name, node, env)
	p.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{
		v1.VolumeMount{
			Name:      p.ObjectMeta.Name + "-config-volume",
			MountPath: "/etc/vppagent-firewall/config.yaml",
			SubPath:   "config.yaml",
		},
	}
	p.Spec.Volumes = []v1.Volume{
		v1.Volume{
			Name: p.ObjectMeta.Name + "-config-volume",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: p.ObjectMeta.Name + "-config-file",
					},
				},
			},
		},
	}
	return p
}

func VppAgentFirewallNSEPod(name string, node *v1.Node, env map[string]string) *v1.Pod {
	ht := new(v1.HostPathType)
	*ht = v1.HostPathDirectoryOrCreate

	var envVars []v1.EnvVar
	for k, v := range env {
		envVars = append(envVars,
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
				containerMod(&v1.Container{
					Name:            "firewall-nse",
					Image:           "networkservicemesh/vppagent-firewall-nse:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
						},
						Requests: nil,
					},
					Env: envVars,
				}),
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
