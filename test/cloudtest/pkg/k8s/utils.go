package k8s

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Utils - basic Kubernetes utils.
type Utils struct {
	config *rest.Config
	clientset *kubernetes.Clientset
}

// NewK8sUtils - Creates a new k8s utils with config file.
func NewK8sUtils (configPath string) (*Utils, error) {
	utils := &Utils{}
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, err
	}

	utils.config = config
	utils.clientset, err = kubernetes.NewForConfig(utils.config)

	return utils, err
}

// GetNodes - return a list of kubernetes nodes.
func (u *Utils) GetNodes() ([]v1.Node, error) {
	nodes, err := u.clientset.CoreV1().Nodes().List(v12.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}