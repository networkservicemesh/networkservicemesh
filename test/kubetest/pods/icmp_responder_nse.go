package pods

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ICMPResponderPod creates a new 'icmp-responder-nse' pod
func ICMPResponderPod(name string, node *v1.Node, env map[string]string, gracePeriod int64,
	dirty, neighbors, routes bool) *v1.Pod {

	envVars := []v1.EnvVar{}
	for k, v := range env {
		envVars = append(envVars,
			v1.EnvVar{
				Name:  k,
				Value: v,
			})
	}

	command := []string{"/bin/icmp-responder-nse"}
	if dirty {
		command = append(command, "-dirty")
	}
	if neighbors {
		command = append(command, "-neighbors")
	}
	if routes {
		command = append(command, "-routes")
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
					Name:            name,
					Image:           "networkservicemesh/test-nse:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
						},
					},
					Env:     envVars,
					Command: command,
				}),
			},
			TerminationGracePeriodSeconds: &gracePeriod,
		},
	}

	if node != nil {
		pod.Spec.NodeSelector = map[string]string{
			"kubernetes.io/hostname": node.Labels["kubernetes.io/hostname"],
		}
	}

	return pod
}
