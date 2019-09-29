package pods

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const name = "spire-proxy"

// SpireProxyPod creates pod that proxy certificate channel from spire-agent
func SpireProxyPod() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: name,
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
		},
		Spec: v1.PodSpec{
			ServiceAccountName: NSMgrServiceAccount,
			Volumes: []v1.Volume{
				spireVolume(),
			},
			Containers: []v1.Container{
				containerMod(&v1.Container{
					Name:            name,
					Image:           "networkservicemesh/test-common:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command:         []string{"/bin/spire-proxy"},
					VolumeMounts: []v1.VolumeMount{
						spireVolumeMount(),
					},
				}),
			},
			TerminationGracePeriodSeconds: &ZeroGraceTimeout,
		},
	}
}
