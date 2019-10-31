package pods

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/utils"
)

const (
	JaegerAPIPort utils.EnvVar = "JAEGER_REST_API_PORT"
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
					Image:           fmt.Sprintf("%v:%v", "jaegertracing/all-in-one:", jaegerVersion),
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
