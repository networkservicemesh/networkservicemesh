package main

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strconv"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func createDNSPatch(tuple *podSpecAndMeta, annotationValue string) (patch []patchOperation) {
	patch = append(patch, addVolume(tuple.spec,
		[]corev1.Volume{{
			Name: "empty-dir-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium:    corev1.StorageMediumDefault,
					SizeLimit: nil,
				},
			},
		}})...)
	patch = append(patch, addVolumeMounts(tuple.spec,
		[]corev1.VolumeMount{{
			ReadOnly:  false,
			Name:      "empty-dir-volume",
			MountPath: "/etc/coredns",
		}})...)
	patch = append(patch, addContainer(tuple.spec,
		[]corev1.Container{
			{
				Name:            "nsm-coredns",
				Image:           "networkservicemesh/nsm-coredns:latest",
				ImagePullPolicy: corev1.PullIfNotPresent,
				Args:            []string{"-conf", "/etc/coredns/Corefile"},
			},
		})...)
	patch = append(patch, addContainer(tuple.spec,
		[]corev1.Container{
			{
				Name:            "nsm-dns-monitor",
				Image:           fmt.Sprintf("%s/%s:%s", getRepo(), "test-common", getTag()),
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/bin/monitoring-dns-nsc"},
				Env: []corev1.EnvVar{{
					Name:  client.AnnotationEnv,
					Value: annotationValue,
				}},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"networkservicemesh.io/socket": resource.NewQuantity(1, resource.DecimalSI).DeepCopy(),
					},
				},
			},
		})...)
	return patch
}

func createNsmInitContainerPatch(annotationValue string) (patch []patchOperation) {
	value := []corev1.Container{{
		Name:  initContainerName,
		Image: fmt.Sprintf("%s/%s:%s", getRepo(), getInitContainer(), getTag()),
		Env: []corev1.EnvVar{{
			Name:  client.AnnotationEnv,
			Value: annotationValue,
		}},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"networkservicemesh.io/socket": resource.MustParse("1"),
			},
		},
	}}

	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  pathInitContainers,
		Value: value,
	})

	return
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

func addVolumeMounts(spec *corev1.PodSpec, added []corev1.VolumeMount) (patch []patchOperation) {
	for i := 0; i < len(spec.Containers); i++ {
		container := &spec.Containers[i]
		path := containersPath + "/" + strconv.Itoa(i) + "/volumeMounts"
		first := len(container.VolumeMounts) == 0
		if !first {
			path = path + "/-"
		}
		for _, v := range added {
			patch = append(patch, patchOperation{
				Op:    "add",
				Path:  path,
				Value: v,
			})
		}
	}
	return patch
}
