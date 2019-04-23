package pods

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
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

func createDefaultDataplaneResources() v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU: resource.NewScaledQuantity(1, -3).DeepCopy(),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU: resource.NewScaledQuantity(1, 0).DeepCopy(),
		},
	}
}
