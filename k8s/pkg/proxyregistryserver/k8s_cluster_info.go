package proxyregistryserver

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type k8sClusterInfo struct {
	clientset *kubernetes.Clientset
}

func NewK8sClusterInfoService(config *rest.Config) (*k8sClusterInfo, error) {
	cs, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	return &k8sClusterInfo{
		clientset: cs,
	}, nil
}

func (k *k8sClusterInfo) GetClusterPublicIP(nodeName string) (string, error) {
	nodes, err := k.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	nodeInternalIp := ""
	nodeExternalIp := ""
	for _, node := range nodes.Items {
		if node.Name == nodeName {
			for _, address := range node.Status.Addresses {
				switch address.Type {
					case "InternalIP":
						nodeInternalIp = address.Address
					case "ExternalIP":
						nodeExternalIp = address.Address
				}
			}
			break
		}
	}

	if len(nodeExternalIp) > 0 {
		return nodeExternalIp, nil
	}
	if len(nodeInternalIp) > 0 {
		return nodeInternalIp, nil
	}

	return "", fmt.Errorf("Node %s was not found", nodeName)
}
