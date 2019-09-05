package pods

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

// DefaultProxyNSMD creates default variables for NSMD.
func DefaultProxyNSMD() map[string]string {
	return map[string]string{}
}

// ProxyNSMgrPod - Proxy NSMgr pod with default configuration
func ProxyNSMgrPod(name string, node *v1.Node, namespace string) *v1.Pod {
	return ProxyNSMgrPodWithConfig(name, node, &NSMgrPodConfig{
		Variables: DefaultProxyNSMD(),
		Namespace: namespace,
	})
}

// ProxyNSMgrPodLiveCheck - Proxy NSMgr pod with default configuration and liveness/readiness probes
func ProxyNSMgrPodLiveCheck(name string, node *v1.Node, namespace string) *v1.Pod {
	return ProxyNSMgrPodWithConfig(name, node, &NSMgrPodConfig{
		liveness:  createProbe("/liveness"),
		readiness: createProbe("/readiness"),
		Variables: DefaultProxyNSMD(),
		Namespace: namespace,
	})
}

// ProxyNSMgrPodWithConfig - Proxy NSMgr pod
func ProxyNSMgrPodWithConfig(name string, node *v1.Node, config *NSMgrPodConfig) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "pnsmgr",
			},
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
			//Kind: "DaemonSet",
		},
		Spec: v1.PodSpec{
			ServiceAccountName: NSMgrServiceAccount,
			Containers: []v1.Container{
				containerMod(&v1.Container{
					Name:            "proxy-nsmd",
					Image:           "networkservicemesh/proxy-nsmd",
					ImagePullPolicy: v1.PullIfNotPresent,
					LivenessProbe:   config.liveness,
					ReadinessProbe:  config.readiness,
					Resources:       createDefaultResources(),
					Ports: []v1.ContainerPort{
						{
							HostPort:      5006,
							ContainerPort: 5006,
						},
					},
				}),
				containerMod(&v1.Container{
					Name:            "proxy-nsmd-k8s",
					Image:           "networkservicemesh/proxy-nsmd-k8s",
					ImagePullPolicy: v1.PullIfNotPresent,
					Resources:       createDefaultResources(),
					Ports: []v1.ContainerPort{
						{
							HostPort:      80,
							ContainerPort: 5005,
						},
					},
				}),
			},
		},
	}

	if insecure, _ := tools.ReadEnvBool("INSECURE", false); insecure {
		if config.Variables == nil {
			config.Variables = map[string]string{}
		}
		config.Variables["INSECURE"] = "true"
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

// ProxyNSMgrSvc - Proxy NSMgr Service pod configuration
func ProxyNSMgrSvc() *v1.Service {
	return &v1.Service{
		ObjectMeta: v12.ObjectMeta{
			Name: "pnsmgr-svc",
			Labels: map[string]string{
				"app": "pnsmgr",
			},
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"app": "pnsmgr",
			},
			Ports: []v1.ServicePort{
				{
					Name:     "pnsmd",
					Protocol: v1.ProtocolTCP,
					Port:     5005,
				},
				{
					Name:     "pnsr",
					Protocol: v1.ProtocolTCP,
					Port:     5006,
				},
			},
		},
	}
}
