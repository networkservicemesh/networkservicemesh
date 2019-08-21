package proxyregistryserver

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/clusterinfo"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	remoteConnection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
)

// InterdomainPlugin is a temporary interface for interdomain plugin, will be removed together with clusterinfo package and usages of it
type InterdomainPlugin interface {
	clusterinfo.ClusterInfoServer
	plugins.RequestPluginServer
	plugins.ConnectionPluginServer
}

type k8sClusterInfo struct {
	clientset *kubernetes.Clientset
}

// NewK8sClusterInfoService creates a ClusterInfoServer
func NewK8sClusterInfoService(config *rest.Config) (InterdomainPlugin, error) {
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &k8sClusterInfo{
		clientset: cs,
	}, nil
}

func (k *k8sClusterInfo) GetNodeIPConfiguration(ctx context.Context, nodeIPConfiguration *clusterinfo.NodeIPConfiguration) (*clusterinfo.NodeIPConfiguration, error) {
	nodes, err := k.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range nodes.Items {
		node := nodes.Items[i]
		nodeInternalIP := ""
		nodeExternalIP := ""

		for _, address := range node.Status.Addresses {
			switch address.Type {
			case v1.NodeInternalIP:
				nodeInternalIP = address.Address
			case v1.NodeExternalIP:
				nodeExternalIP = address.Address
			}
		}

		if node.Name == nodeIPConfiguration.NodeName ||
			len(nodeInternalIP) > 0 && nodeInternalIP == nodeIPConfiguration.InternalIP ||
			len(nodeExternalIP) > 0 && nodeExternalIP == nodeIPConfiguration.ExternalIP {

			return &clusterinfo.NodeIPConfiguration{
				NodeName:   node.Name,
				ExternalIP: nodeExternalIP,
				InternalIP: nodeInternalIP,
			}, nil
		}
	}

	return nil, fmt.Errorf("node was not found: %v", nodeIPConfiguration)
}

func (k *k8sClusterInfo) UpdateRequest(ctx context.Context, wrapper *plugins.RequestWrapper) (*plugins.RequestWrapper, error) {
	for _, mechanism := range wrapper.GetRequest().GetRequestMechanismPreferences() {
		switch mechanism.GetMechanismType() {
		case remoteConnection.MechanismType_VXLAN:
			ip := mechanism.GetParameters()[remoteConnection.VXLANSrcIP]

			ipConfig, err := k.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{InternalIP: ip})
			if err == nil {
				mechanism.GetParameters()[remoteConnection.VXLANSrcExtIP] = ipConfig.ExternalIP
			}
		}
	}
	return wrapper, nil
}

func (k *k8sClusterInfo) UpdateConnection(ctx context.Context, wrapper *plugins.ConnectionWrapper) (*plugins.ConnectionWrapper, error) {
	mechanism := wrapper.GetConnection().GetConnectionMechanism()
	switch mechanism.GetMechanismType() {
	case remoteConnection.MechanismType_VXLAN:
		ip := mechanism.GetParameters()[remoteConnection.VXLANDstIP]

		ipConfig, err := k.GetNodeIPConfiguration(ctx, &clusterinfo.NodeIPConfiguration{InternalIP: ip})
		if err == nil {
			mechanism.GetParameters()[remoteConnection.VXLANDstExtIP] = ipConfig.ExternalIP
		}
	}
	return wrapper, nil
}

func (k *k8sClusterInfo) ValidateConnection(ctx context.Context, wrapper *plugins.ConnectionWrapper) (*plugins.ConnectionValidationResult, error) {
	return &plugins.ConnectionValidationResult{Status: plugins.ConnectionValidationStatus_SUCCESS}, nil
}
