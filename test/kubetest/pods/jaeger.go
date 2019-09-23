package pods

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func JaegerService(pod *v1.Pod) *v1.Service {
	return &v1.Service{
		ObjectMeta: v12.ObjectMeta{
			Name:   pod.Name,
			Labels: map[string]string{"run": pod.Name},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: "http", Port: 16686, Protocol: "TCP"},
				{Name: "jaeger", Port: 6831, Protocol: "UDP"},
			},
			Selector: map[string]string{"run": "jaeger"},
		},
	}
}
func Jaeger() *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: "jaeger",
			Labels: map[string]string{
				"run": "jaeger",
			},
		},
		TypeMeta: v12.TypeMeta{
			Kind: "Deployment",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "jaeger",
					Image:           "jaegertracing/all-in-one:latest",
					ImagePullPolicy: v1.PullIfNotPresent,
					Ports: []v1.ContainerPort{
						{Name: "http", ContainerPort: 16686, Protocol: "TCP"},
						{Name: "jaeger", ContainerPort: 6831, Protocol: "UDP"},
					},
				},
			},
		},
	}
	return pod
}

func newJaegerEnvVar() []v1.EnvVar {
	jaegerHost := "jaeger.nsm-system"
	jaegerPort := "6831"

	if value := os.Getenv("JAEGER_AGENT_HOST"); value != "" {
		jaegerHost = value
	}
	if value := os.Getenv("JAEGER_AGENT_PORT"); value != "" {
		jaegerPort = value
	}
	return []v1.EnvVar{
		{
			Name:  "JAEGER_AGENT_HOST",
			Value: jaegerHost,
		},
		{
			Name:  "JAEGER_AGENT_PORT",
			Value: jaegerPort,
		},
		{
			Name:  "TRACER_ENABLED",
			Value: "true",
		},
	}
}
