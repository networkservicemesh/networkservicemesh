package registryserver

import (
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha3"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/prefixcollector"
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
	logrus.Info("ClusterConfiguration request")
	kubeadmConfig, err := k.clientset.CoreV1().
		ConfigMaps("kube-system").
		Get("kubeadm-config", metav1.GetOptions{})
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	clusterConfiguration := &v1alpha3.ClusterConfiguration{}
	err = yaml.NewYAMLOrJSONDecoder(strings.NewReader(kubeadmConfig.Data["ClusterConfiguration"]), 4096).
		Decode(clusterConfiguration)
	if err != nil {
		logrus.Error(err)
	}

	return &registry.ClusterConfiguration{
		PodSubnet:     clusterConfiguration.Networking.PodSubnet,
		ServiceSubnet: clusterConfiguration.Networking.ServiceSubnet,
	}, nil
}

func (k *k8sClusterInfo) MonitorSubnets(empty *empty.Empty, stream registry.ClusterInfo_MonitorSubnetsServer) error {
	logrus.Info("MonitorSubnets request")
	pw, err := prefixcollector.WatchPodCIDR(k.clientset)
	if err != nil {
		return err
	}
	defer pw.Stop()

	sw, err := prefixcollector.WatchServiceIpAddr(k.clientset)
	if err != nil {
		return err
	}
	defer sw.Stop()

	for {
		select {
		case <-stream.Context().Done():
			logrus.Infof("MonitorSubnets deadline exceeded")
			return stream.Context().Err()
		case podSubnet := <-pw.ResultChan():
			err := stream.Send(&registry.SubnetExtendingResponse{
				Type:   registry.SubnetExtendingResponse_POD,
				Subnet: podSubnet.String(),
			})
			if err != nil {
				logrus.Error(err)
				return err
			}
		case serviceSubnet := <-sw.ResultChan():
			err := stream.Send(&registry.SubnetExtendingResponse{
				Type:   registry.SubnetExtendingResponse_SERVICE,
				Subnet: serviceSubnet.String(),
			})
			if err != nil {
				logrus.Error(err)
				return err
			}
		}
	}
}
