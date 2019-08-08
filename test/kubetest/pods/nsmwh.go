package pods

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NSWHPod(name string, node *v1.Node, env map[string]string) *v1.Pod {
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
			Labels: map[string]string{
				"app": "nsm-admission-webhook",
			},
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				containerMod(&v1.Container{
					Name:            name,
					Image:           "networkservicemesh/nsmwh:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					Env:             envVars,
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "webhook-certs",
							MountPath: "/etc/webhook/certs",
							ReadOnly:  true,
						},
					},
				}),
			},
			Volumes: []v1.Volume{
				{
					Name: "webhook-certs",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "nsm-admission-webhook-certs",
						},
					},
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
