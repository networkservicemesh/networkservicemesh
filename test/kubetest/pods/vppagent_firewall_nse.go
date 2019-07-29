package pods

import (
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VppAgentFirewallNSEConfigMapICMPHTTP creates a new 'vppagent-firewall-nse' config map
func VppAgentFirewallNSEConfigMapICMPHTTP(name, namespace string) *v1.ConfigMap {
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

// VppAgentFirewallNSEPodWithConfigMap creates a new 'vppagent-firewall-nse' pod with config map set
func VppAgentFirewallNSEPodWithConfigMap(name string, node *v1.Node, env map[string]string) *v1.Pod {
	p := VppAgentFirewallNSEPod(name, node, env)
	p.Spec.Containers[0].VolumeMounts = append(p.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
		Name:      p.ObjectMeta.Name + "-config-volume",
		MountPath: "/etc/vppagent-firewall/config.yaml",
		SubPath:   "config.yaml",
	})
	p.Spec.Volumes = append(p.Spec.Volumes, v1.Volume{
		Name: p.ObjectMeta.Name + "-config-volume",
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: p.ObjectMeta.Name + "-config-file",
				},
			},
		},
	})
	return p
}

// VppAgentFirewallNSEPod creates a new 'vppagent-firewall-nse' pod
func VppAgentFirewallNSEPod(name string, node *v1.Node, env map[string]string) *v1.Pod {
	return VppTestCommonPod("vppagent-firewall-nse", name, "firewall-nse", node, env, NSEServiceAccount)
}
