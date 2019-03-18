package registryserver

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha3"
	"strings"
)

type k8sClusterInfo struct {
	clientset *kubernetes.Clientset
}

func NewK8sClusterInfoService(config *rest.Config) (registry.ClusterInfoServer, error) {
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &k8sClusterInfo{
		clientset: cs,
	}, nil
}

func (k *k8sClusterInfo) GetClusterConfiguration(ctx context.Context, in *empty.Empty) (*registry.ClusterConfiguration, error) {
	kubeadmConfig, err := k.clientset.CoreV1().
		ConfigMaps("kube-system").
		Get("kubeadm-config", metav1.GetOptions{})

	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	clusterConfiguration := &v1alpha3.ClusterConfiguration{}
	err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(kubeadmConfig.Data["ClusterConfiguration"]), 1000).
		Decode(clusterConfiguration)
	if err != nil {
		logrus.Error(err)
	}

	return &registry.ClusterConfiguration{
		PodSubnet:     clusterConfiguration.Networking.PodSubnet,
		ServiceSubnet: clusterConfiguration.Networking.ServiceSubnet,
	}, nil
}
