package k8s

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sUtils struct {
	config *rest.Config
	clientset *kubernetes.Clientset
}

func NewK8sUtils (configPath string) (*K8sUtils, error) {
	utils := &K8sUtils{}
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, err
	}

	utils.config = config
	utils.clientset, err = kubernetes.NewForConfig(utils.config)

	return utils, err
}

func (u *K8sUtils) GetNodes() ([]v1.Node, error) {
	nodes, err := u.clientset.CoreV1().Nodes().List(v12.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}