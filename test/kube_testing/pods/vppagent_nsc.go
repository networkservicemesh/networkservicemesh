package pods

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func VppagentNSC(name string, node *v1.Node, env map[string]string) *v1.Pod {
	envVars := []v1.EnvVar{}
	for k, v := range env {
		envVars = append(envVars, v1.EnvVar{
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
			HostPID: true,
			Containers: []v1.Container{
				{
					Name:            "vppagent-nsc",
					Image:           "networkservicemesh/vppagent-nsc:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					Env:             envVars,
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
						},
					},
				},
			},
			TerminationGracePeriodSeconds: &ZeroGraceTimeout,
		},
	}
	return pod
}
