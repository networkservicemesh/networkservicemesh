package pods

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AdmissionWebhookDeployment returns deployment named `name` which starts container from `image`
func AdmissionWebhookDeployment(name, image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "nsm-admission-webhook",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nsm-admission-webhook"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nsm-admission-webhook",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						containerMod(&v1.Container{
							Name:            name,
							Image:           image,
							ImagePullPolicy: v1.PullIfNotPresent,
							Env: []v1.EnvVar{
								{
									Name:  "TAG",
									Value: containerTag,
								},
								{
									Name:  "REPO",
									Value: containerRepo,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "webhook-certs",
									MountPath: "/etc/webhook/certs",
									ReadOnly:  true,
								},
							},
							LivenessProbe:  createProbe("/liveness"),
							ReadinessProbe: createProbe("/readiness"),
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
				},
			},
		},
	}
}
