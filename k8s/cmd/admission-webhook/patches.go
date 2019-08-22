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

func createDNSPatch(tuple *podSpecAndMeta) (patch []patchOperation) {
	patch = append(patch, addContainer(tuple.spec,
		[]corev1.Container{
			{
				Name:            "nsm-coredns",
				Image:           fmt.Sprintf("%s/%s:%s", getRepo(), "nsm-coredns", getTag()),
				ImagePullPolicy: corev1.PullIfNotPresent,
				Args:            []string{"-conf", "/etc/coredns/Corefile"},
				VolumeMounts: []corev1.VolumeMount{{
					ReadOnly:  false,
					Name:      "nsm-coredns-volume",
					MountPath: "/etc/coredns",
				}},
			},
		})...)
	patch = append(patch, addContainer(tuple.spec,
		[]corev1.Container{
			{
				Name:            "nsm-dns-monitor",
				Image:           fmt.Sprintf("%s/%s:%s", getRepo(), "nsm-monitor", getTag()),
				ImagePullPolicy: corev1.PullIfNotPresent,
				VolumeMounts: []corev1.VolumeMount{{
					ReadOnly:  false,
					Name:      "nsm-coredns-volume",
					MountPath: "/etc/coredns",
				}},
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
	patch = append(patch, replaceDNSConfig()...)
	return patch
}

func createNsmInitContainerPatch(annotationValue string) []patchOperation {
	var patch []patchOperation

	envVals := []corev1.EnvVar{{
		Name:  client.AnnotationEnv,
		Value: annotationValue,
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
	jaegerPort := getJaegerPort()
	if jaegerPort != "" {
		envVals = append(envVals,
			corev1.EnvVar{
				Name:  jaegerPortEnv,
				Value: jaegerPort,
			})
	}

	value := []corev1.Container{{
		Name:            initContainerName,
		Image:           fmt.Sprintf("%s/%s:%s", getRepo(), getInitContainer(), getTag()),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVals,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"networkservicemesh.io/socket": resource.MustParse("1"),
			},
		},
	}}

	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  initContainersPath,
		Value: value,
	})

	return patch
}

func replaceDNSConfig() []patchOperation {
	return []patchOperation{{
		Op:   "replace",
		Path: dnsConfigPath,
		Value: &corev1.PodDNSConfig{
			Nameservers: []string{"127.0.0.1"},
			Searches:    []string{"default.svc.cluster.local", "svc.cluster.local", "cluster.local"},
		},
	}}
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
