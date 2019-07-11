package pods

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VPNGatewayNSEPod creates a new 'vpn-gateway-nse' pod
func VPNGatewayNSEPod(name string, node *v1.Node, env map[string]string) *v1.Pod {
	ht := new(v1.HostPathType)
	*ht = v1.HostPathDirectoryOrCreate

	envVars := []v1.EnvVar{}
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
			ServiceAccountName: NSEServiceAccount,
			Volumes: []v1.Volume{
				{
					Name: "spire-agent-socket",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/run/spire/sockets",
							Type: ht,
						},
					},
				},
			},
			Containers: []v1.Container{
				containerMod(&v1.Container{
					Name:            "vpn-gateway",
					Image:           "networkservicemesh/test-common:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command: []string{
						"/bin/icmp-responder-nse",
					},
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
						},
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "spire-agent-socket",
							MountPath: "/run/spire/sockets",
							ReadOnly:  true,
						},
					},
					Env: envVars,
				}),
				{
					Name:  "nginx",
					Image: "networkservicemesh/nginx",
				},
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
