package registryserver

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	attempts = 10
	attemptDelay = 100 * time.Millisecond
)

type nsmRegistryService struct {
	nsmName string
	cache   RegistryCache
}

func newNsmRegistryService(nsmName string, cache RegistryCache) *nsmRegistryService {
	return &nsmRegistryService{
		nsmName: nsmName,
		cache:   cache,
	}
}

func (n *nsmRegistryService) RegisterNSM(ctx context.Context, nsm *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	logrus.Infof("Received RegisterNSM(%v)", nsm)

	cachedValue := n.cache.GetNetworkServiceManager(nsm.GetName())
	logrus.Infof("Cached value %v", cachedValue)
	if nsm.GetName() == "" || cachedValue == nil {
		return n.create(nsm)
	}

	return n.update(nsm)
}

func (n *nsmRegistryService) GetEndpoints(context.Context, *empty.Empty) (*registry.NetworkServiceEndpointList, error) {
	logrus.Info("Received GetEndpoints")

	endpoints := n.cache.GetEndpointsByNsm(n.nsmName)
	var response []*registry.NetworkServiceEndpoint
	for _, endpoint := range endpoints {
		ns, err := n.cache.GetNetworkService(endpoint.Spec.NetworkServiceName)
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		response = append(response, mapNseFromCustomResource(endpoint, ns.Spec.Payload))
	}

	return &registry.NetworkServiceEndpointList{
		NetworkServiceEndpoints: response,
	}, nil
}

func (n *nsmRegistryService) create(nsm *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	nsmCr := mapNsmToCustomResource(nsm)
	nsmCr.SetName(n.nsmName)

	nsmCr, err := n.cache.AddNetworkServiceManager(nsmCr)
	if err != nil {
		logrus.Errorf("Failed to create nsm: %s", err)
		return nil, err
	}

	return mapNsmFromCustomResource(nsmCr), nil
}

func (n *nsmRegistryService) update(nsm *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	if nsm.GetName() != n.nsmName {
		return nil, fmt.Errorf("wrong nsm name %v, expected - %v", nsm.GetName(), n.nsmName)
	}

	oldNsm := n.cache.GetNetworkServiceManager(nsm.Name)
	if oldNsm == nil {
		return nil, fmt.Errorf("no nsm with name %v", nsm.Name)
	}

	nsmCr := mapNsmToCustomResource(nsm)

	nsmCr.ObjectMeta = oldNsm.ObjectMeta
	logrus.Infof("Performing update: %v", nsmCr)
	var err error

	//TODO: Workaround for issue: https://github.com/kubernetes/kubernetes/issues/71139
	for i := 0; i < attempts; i++ {
		nsmCr, err = n.cache.UpdateNetworkServiceManager(nsmCr)
		if err != nil {
			logrus.Errorf("Failed to update nsm: %s retrying...", err)
			if i == attempts-1 {
				return nil, err
			} else {
				logrus.Errorf("Waiting for Kubernetes to be consistent..")
				<- time.Tick(attemptDelay)
				continue
			}
		}
		break
	}

	return mapNsmFromCustomResource(nsmCr), nil
}
