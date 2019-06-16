package k8s

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type KubernetesValidator interface {
	Validate() error
}

type ValidationFactory interface {
	CreateValidator(config *config.ClusterProviderConfig, location string) (KubernetesValidator, error)
}

type k8sFactory struct {
}

type k8sValidator struct {
	config   *config.ClusterProviderConfig
	location string
	utils    *K8sUtils
}

func (v *k8sValidator) Validate() error {
	requiedNodes := v.config.NodeCount
	nodes, err := v.utils.GetNodes()
	if err != nil {
		return err
	}
	if len(nodes) >= requiedNodes {
		return nil
	}
	msg := fmt.Sprintf("Cluster doesn't have required number of nodes to be available. Required: %v Available: %v\n", requiedNodes, len(nodes))
	logrus.Errorf(msg)
	err = fmt.Errorf(msg)
	return err
}

func (*k8sFactory) CreateValidator(config *config.ClusterProviderConfig, location string) (KubernetesValidator, error) {
	utils, err := NewK8sUtils(location)
	if err != nil {
		return nil, err
	}

	return &k8sValidator{
		config:   config,
		location: location,
		utils:    utils,
	}, nil
}

func CreateFactory() ValidationFactory {
	return &k8sFactory{}
}
