package main

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/networkservicemesh/networkservicemesh/sdk/client"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func createDNSPatch(tuple *podSpecAndMeta, annotationValue string) (patch []patchOperation) {
	// TODO: now order of containers is important since nsmdp assign proper workspace only to the first container
	patch = append(patch, addContainer(tuple.spec,
		[]corev1.Container{
			{
				Name:            "nsm-dns-monitor",
				Command:         []string{"/bin/nsm-monitor"},
				Image:           fmt.Sprintf("%s/%s:%s", getRepo(), "nsm-monitor", getTag()),
				ImagePullPolicy: corev1.PullIfNotPresent,
				Env: []corev1.EnvVar{
					{
						Name:  "MONITOR_DNS_CONFIGS",
						Value: "true",
					},
					{
						Name:  client.AnnotationEnv,
						Value: annotationValue,
					},
				},
				VolumeMounts: []corev1.VolumeMount{{
					ReadOnly:  false,
					Name:      "nsm-coredns-volume",
					MountPath: "/etc/coredns",
				}},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"networkservicemesh.io/socket": resource.MustParse("1"),
					},
				},
			},
		})...)
	patch = append(patch, addContainer(tuple.spec,
		[]corev1.Container{
			{
				Name:            "coredns",
				Image:           "networkservicemesh/coredns:master",
				ImagePullPolicy: corev1.PullIfNotPresent,
				Args:            []string{"-conf", "/etc/coredns/Corefile"},
				VolumeMounts: []corev1.VolumeMount{{
					ReadOnly:  false,
					Name:      "nsm-coredns-volume",
					MountPath: "/etc/coredns",
				}},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"networkservicemesh.io/socket": resource.MustParse("1"),
					},
				},
			},
		})...)

	patch = append(patch, addVolume(tuple.spec,
		[]corev1.Volume{{
			Name: "nsm-coredns-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium:    corev1.StorageMediumDefault,
					SizeLimit: nil,
				},
			},
		}})...)
	return patch
}

func createNsmInitContainerPatch(target []corev1.Container, annotationValue string) []patchOperation {
	var patch []patchOperation

	namespace := getNamespace()
	envVals := []corev1.EnvVar{
		{
			Name:  client.AnnotationEnv,
			Value: annotationValue,
		},
		{
			Name:  client.NamespaceEnv,
			Value: namespace,
		},
	}
	jaegerHost := getJaegerHost()
	if jaegerHost != "" {
		envVals = append(envVals,
			corev1.EnvVar{
				Name:  jaegerHostEnv,
				Value: jaegerHost,
			})
	}
	tracerEnabled := getTracerEnabled()
	if tracerEnabled != "" {
		envVals = append(envVals,
			corev1.EnvVar{
				Name:  tracerEnabledEnv,
				Value: tracerEnabled,
			})
	}

	jaegerPort := getJaegerPort()
	if jaegerPort != "" {
		envVals = append(envVals,
			corev1.EnvVar{
				Name:  jaegerPortEnv,
				Value: jaegerPort,
			})
	}
	var value interface{}
	nsmInitContainer := corev1.Container{
		Name:            initContainerName,
		Image:           fmt.Sprintf("%s/%s:%s", getRepo(), getInitContainer(), getTag()),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVals,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"networkservicemesh.io/socket": resource.MustParse("1"),
			},
		},
	}
	dnsNsmInitContainer := corev1.Container{
		Name:            dnsInitContainerDefault,
		Image:           fmt.Sprintf("%s/%s:%s", getRepo(), dnsInitContainerDefault, getTag()),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVals,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"networkservicemesh.io/socket": resource.MustParse("1"),
			},
		},
		Command: []string{"/bin/" + dnsInitContainerDefault},
		VolumeMounts: []corev1.VolumeMount{{
			ReadOnly:  false,
			Name:      "nsm-coredns-volume",
			MountPath: "/etc/coredns",
		}},
	}
	value = append([]corev1.Container{dnsNsmInitContainer, nsmInitContainer}, target...)

	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  initContainersPath,
		Value: value,
	})

	return patch
}

func addVolume(spec *corev1.PodSpec, added []corev1.Volume) (patch []patchOperation) {
	first := len(spec.Volumes) == 0
	var value interface{}
	for i := 0; i < len(added); i++ {
		value = added[i]
		path := volumePath
		if first {
			first = false
			value = []corev1.Volume{added[i]}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func addContainer(spec *corev1.PodSpec, containers []corev1.Container) (patch []patchOperation) {
	first := len(spec.Containers) == 0
	for i := 0; i < len(containers); i++ {
		value := &containers[i]
		path := containersPath
		if first {
			first = false
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}

	return patch
}
