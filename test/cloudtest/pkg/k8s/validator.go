package k8s

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
	"github.com/sirupsen/logrus"
)

// KubernetesValidator - a validator to check periodically of cluster livenes.
type KubernetesValidator interface {
	Validate() error
}

// ValidationFactory - factory to create validator
type ValidationFactory interface {
	// CreateValidator - return intanceof of validator with config and cluster config
	CreateValidator(config *config.ClusterProviderConfig, location string) (KubernetesValidator, error)
}

type k8sFactory struct {
}

type k8sValidator struct {
	config   *config.ClusterProviderConfig
	location string
	utils    *Utils
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
// CreateFactory - creates a validation factory.
func CreateFactory() ValidationFactory {
	return &k8sFactory{}
}
