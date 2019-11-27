package pods

import (
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/prefix_pool"
)

const (
	configVolumeName = "nsm-config-volume"
)

var ZeroGraceTimeout int64 = 0

func createProbe(path string) *v1.Probe {
	return &v1.Probe{
		Handler: v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path:   path,
				Port:   intstr.IntOrString{Type: 0, IntVal: 5555, StrVal: ""},
				Scheme: "HTTP",
			},
		},
		InitialDelaySeconds: 1,
		PeriodSeconds:       3,
		TimeoutSeconds:      10,
	}
}

func createDefaultResources() v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU: resource.NewScaledQuantity(1, -3).DeepCopy(),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU: resource.NewScaledQuantity(1, 0).DeepCopy(),
		},
	}
}

func createDefaultForwarderResources() v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU: resource.NewScaledQuantity(1, -3).DeepCopy(),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU: resource.NewScaledQuantity(1, 0).DeepCopy(),
		},
	}
}

func nsmConfigVolume() v1.Volume {
	return v1.Volume{
		Name: configVolumeName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "nsm-config",
				},
			},
		},
	}
}

func nsmConfigVolumeMount() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      configVolumeName,
		MountPath: prefix_pool.NsmConfigDir,
	}
}

func spireVolume() v1.Volume {
	ht := v1.HostPathDirectoryOrCreate

	return v1.Volume{
		Name: "spire-agent-socket",
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: "/run/spire/sockets",
				Type: &ht,
			},
		},
	}
}

func spireVolumeMount() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      "spire-agent-socket",
		MountPath: "/run/spire/sockets",
		ReadOnly:  true,
	}
}

func setInsecureEnvIfExist(envs map[string]string) map[string]string {
	insecure, _ := tools.IsInsecure()
	if !insecure {
		return envs
	}

	if envs == nil {
		envs = map[string]string{}
	}

	envs[tools.InsecureEnv] = strconv.FormatBool(true)
	return envs
}
