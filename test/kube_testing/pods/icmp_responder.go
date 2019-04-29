package pods

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ICMPResponderPod(name string, node *v1.Node, env map[string]string) *v1.Pod {
	ht := new(v1.HostPathType)
	*ht = v1.HostPathDirectoryOrCreate

	nsc_container := containerMod(&v1.Container{
		Name:            "icmp-responder-nse",
		Image:           "networkservicemesh/icmp-responder-nse:latest",
		ImagePullPolicy: v1.PullIfNotPresent,
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
			},
			Requests: nil,
		},
	})
	for k, v := range env {
		nsc_container.Env = append(nsc_container.Env,
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
			Containers: []v1.Container{nsc_container},
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
