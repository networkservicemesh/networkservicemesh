package prefixcollector

import (
	"fmt"
	"strings"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha3"
)

func getExcludedPrefixesChan(clientset *kubernetes.Clientset) (<-chan prefix_pool.PrefixPool, error) {
	// trying to get excludePrefixes from kubeadm-config, if it exists
	if configMapPrefixes, err := getExcludedPrefixesFromConfigMap(clientset); err == nil {
		poolCh := make(chan prefix_pool.PrefixPool, 1)
		pool, err := prefix_pool.NewPrefixPool(configMapPrefixes...)
		if err != nil {
			return nil, err
		}
		poolCh <- pool
		return poolCh, nil
	}

	// seems like we don't have kubeadm-config in cluster, starting monitor client
	return monitorSubnets(clientset), nil
}

func getExcludedPrefixesFromConfigMap(clientset *kubernetes.Clientset) ([]string, error) {
	kubeadmConfig, err := clientset.CoreV1().
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
		return nil, err
	}

	podSubnet := clusterConfiguration.Networking.PodSubnet
	serviceSubnet := clusterConfiguration.Networking.ServiceSubnet

	if podSubnet == "" {
		return nil, fmt.Errorf("clusterConfiguration.Networking.PodSubnet is empty")
	}
	if serviceSubnet == "" {
		return nil, fmt.Errorf("clusterConfiguration.Networking.ServiceSubnet is empty")
	}

	return []string{
		podSubnet,
		serviceSubnet,
	}, nil
}

func monitorSubnets(clientset *kubernetes.Clientset) <-chan prefix_pool.PrefixPool {
	logrus.Infof("Start monitoring prefixes to exclude")
	poolCh := make(chan prefix_pool.PrefixPool, 1)

	go func() {
		for {
			errCh := make(chan error)
			go monitorReservedSubnets(poolCh, errCh, clientset)
			err := <-errCh
			logrus.Error(err)
		}
	}()

	return poolCh
}

func monitorReservedSubnets(poolCh chan prefix_pool.PrefixPool, errCh chan<- error, clientset *kubernetes.Clientset) {
	pw, err := WatchPodCIDR(clientset)
	if err != nil {
		errCh <- err
		return
	}
	defer pw.Stop()

	sw, err := WatchServiceIpAddr(clientset)
	if err != nil {
		errCh <- err
		return
	}
	defer sw.Stop()

	var podSubnet, serviceSubnet string
	for {
		select {
		case subnet := <-pw.ResultChan():
			podSubnet = subnet.String()
		case subnet := <-sw.ResultChan():
			serviceSubnet = subnet.String()
		}
		sendPrefixPool(poolCh, podSubnet, serviceSubnet)
	}
}

func sendPrefixPool(poolCh chan prefix_pool.PrefixPool, podSubnet, serviceSubnet string) {
	pool, err := getPrefixPool(podSubnet, serviceSubnet)
	if err != nil {
		logrus.Errorf("Failed to create a prefix pool: %v", err)
		return
	}
	select {
	case <-poolCh:
	default:
	}
	poolCh <- pool
}

func getPrefixPool(podSubnet, serviceSubnet string) (prefix_pool.PrefixPool, error) {
	var prefixes []string
	if len(podSubnet) > 0 {
		prefixes = append(prefixes, podSubnet)
	}
	if len(serviceSubnet) > 0 {
		prefixes = append(prefixes, serviceSubnet)
	}

	pool, err := prefix_pool.NewPrefixPool(prefixes...)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
