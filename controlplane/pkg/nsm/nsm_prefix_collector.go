package nsm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/sirupsen/logrus"
)

func GetExcludedPrefixes(serviceRegistry serviceregistry.ServiceRegistry) (<-chan prefix_pool.PrefixPool, error) {
	excludePrefixes := getExcludePrefixesFromEnv()

	// trying to get excludePrefixes from kubeadm-config, if it exists
	if configMapPrefixes, err := getExcludedPrefixesFromConfigMap(serviceRegistry); err == nil {
		poolCh := make(chan prefix_pool.PrefixPool, 1)
		pool, err := prefix_pool.NewPrefixPool(append(excludePrefixes, configMapPrefixes...)...)
		if err != nil {
			return nil, err
		}
		poolCh <- pool
		return poolCh, nil
	}

	// seems like we don't have kubeadm-config in cluster, starting monitor client
	return monitorReservedSubnets(serviceRegistry, excludePrefixes...), nil
}

func getExcludedPrefixesFromConfigMap(serviceRegistry serviceregistry.ServiceRegistry) ([]string, error) {
	clusterInfoClient, err := serviceRegistry.ClusterInfoClient()
	if err != nil {
		return nil, fmt.Errorf("error during ClusterInfoClient creation: %v", err)
	}

	clusterConfiguration, err := clusterInfoClient.GetClusterConfiguration(context.Background(), &empty.Empty{})
	if err != nil {
		return nil, fmt.Errorf("error during GetClusterConfiguration request: %v", err)
	}

	if clusterConfiguration.PodSubnet == "" {
		return nil, fmt.Errorf("clusterConfiguration.PodSubnet is empty")
	}

	if clusterConfiguration.ServiceSubnet == "" {
		return nil, fmt.Errorf("clusterConfiguration.ServiceSubnet is empty")
	}

	return []string{
		clusterConfiguration.PodSubnet,
		clusterConfiguration.ServiceSubnet,
	}, nil
}

func getExcludePrefixesFromEnv() []string {
	excludedPrefixesEnv, ok := os.LookupEnv(nsmd.ExcludedPrefixesEnv)
	if !ok {
		return []string{}
	}
	logrus.Infof("Getting excludedPrefixes from ENV: %v", excludedPrefixesEnv)
	return strings.Split(excludedPrefixesEnv, ",")
}

const (
	poolChannelSize = 10
)

func monitorReservedSubnets(s serviceregistry.ServiceRegistry, additionalPrefixes ...string) <-chan prefix_pool.PrefixPool {
	logrus.Infof("Start monitoring prefixes to exclude")
	poolCh := make(chan prefix_pool.PrefixPool, poolChannelSize)

	go func() {
		for {
			errCh := make(chan error, 1)
			listener := &subnetStreamListener{
				errCh:              errCh,
				outCh:              poolCh,
				additionalPrefixes: additionalPrefixes,
				serviceRegistry:    s,
			}
			go listener.listen()
			err := <-errCh
			logrus.Error(err)
		}
	}()

	return poolCh
}

type subnetStreamListener struct {
	errCh              chan<- error
	outCh              chan<- prefix_pool.PrefixPool
	serviceRegistry    serviceregistry.ServiceRegistry
	additionalPrefixes []string
}

func (l *subnetStreamListener) listen() {
	clusterInfoClient, err := l.serviceRegistry.ClusterInfoClient()
	if err != nil {
		l.errCh <- fmt.Errorf("error during ClusterInfoClient creation: %v", err)
		return
	}

	stream, err := clusterInfoClient.MonitorSubnets(context.Background(), &empty.Empty{})
	if err != nil {
		l.errCh <- fmt.Errorf("error during ClusterInfoClient.MonitorSubnet calling: %v", err)
		return
	}

	var podSubnet string
	var serviceSubnet string

	for {
		extendResponse, err := stream.Recv()
		if err != nil {
			l.errCh <- err
			return
		}
		logrus.Infof("Received subnetExtendResponse: %v", extendResponse)

		if extendResponse.Type == registry.SubnetExtendingResponse_POD {
			podSubnet = extendResponse.Subnet
		}

		if extendResponse.Type == registry.SubnetExtendingResponse_SERVICE {
			serviceSubnet = extendResponse.Subnet
		}

		prefixes := l.additionalPrefixes
		if len(podSubnet) > 0 {
			prefixes = append(prefixes, podSubnet)
		}
		if len(serviceSubnet) > 0 {
			prefixes = append(prefixes, serviceSubnet)
		}

		pool, err := prefix_pool.NewPrefixPool(prefixes...)

		if err != nil {
			logrus.Error(err)
			continue
		}

		l.outCh <- pool
	}
}
