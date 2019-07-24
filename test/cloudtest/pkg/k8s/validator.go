package k8s

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/cloudtest/pkg/config"
)

// KubernetesValidator - a validator to check periodically of cluster livenes.
type KubernetesValidator interface {
	Validate() error
	WaitValid(context context.Context) error
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

func (v *k8sValidator) WaitValid(context context.Context) error {
	for {
		err := v.Validate()
		if err == nil {
			break
		}
		// Waiting a bit.
		select {
		case <-time.After(1 * time.Second):
		case <-context.Done():
			return err
		}
	}
	return nil
}

func isNodeReady(node *v1.Node) bool {
	conditions := node.Status.Conditions
	for idx := range conditions {
		if conditions[idx].Type == v1.NodeReady {
			resultValue := conditions[idx].Status == v1.ConditionTrue
			return resultValue
		}
	}
	return false
}
func (v *k8sValidator) Validate() error {
	requiedNodes := v.config.NodeCount
	nodes, err := v.utils.GetNodes()
	if err != nil {
		return err
	}

	ready := 0
	for idx := range nodes {
		if isNodeReady(&nodes[idx]) {
			ready++
		}
	}
	if ready >= requiedNodes {
		return nil
	}
	msg := fmt.Sprintf("Cluster doesn't have required number of nodes to be available. Required: %v Available: %v\n", requiedNodes, len(nodes))
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
