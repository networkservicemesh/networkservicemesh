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

func Jaeger(node *v1.Node) *v1.Pod {
	nodeName := ""
	if node != nil {
		nodeName = fmt.Sprintf("-%v", node.Name)
	}
	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: fmt.Sprintf("jaeger%v", nodeName),
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
	pod.Spec.NodeSelector = map[string]string{
		"kubernetes.io/hostname": node.Labels["kubernetes.io/hostname"],
	}
	return pod
}
