package registryserver

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/sirupsen/logrus"
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

	if nsm.GetName() == "" || n.cache.GetNetworkServiceManager(nsm.GetName()) == nil {
		return n.create(nsm)
	}

	return n.update(nsm)
}

func (n *nsmRegistryService) GetEndpoints(context.Context, *empty.Empty) (*registry.NetworkServiceEndpointList, error) {
	logrus.Infof("Received GetEndpoints(%v)")

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
	nsmCdr := mapNsmToCustomResource(nsm)
	nsmCdr.SetName(n.nsmName)

	nsmCdr, err := n.cache.AddNetworkServiceManager(nsmCdr)
	if err != nil {
		logrus.Errorf("Failed to create nsm: %s", err)
		return nil, err
	}

	return mapNsmFromCustomResource(nsmCdr), nil
}

func (n *nsmRegistryService) update(nsm *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	if nsm.GetName() != n.nsmName {
		return nil, fmt.Errorf("wrong nsm name %v, expected - %v", nsm.GetName(), n.nsmName)
	}

	oldNsm := n.cache.GetNetworkServiceManager(nsm.Name)
	if oldNsm == nil {
		return nil, fmt.Errorf("no nsm with name %v", nsm.Name)
	}

	nsmCdr := mapNsmToCustomResource(nsm)
	nsmCdr.ObjectMeta = oldNsm.ObjectMeta

	nsmCdr, err := n.cache.UpdateNetworkServiceManager(nsmCdr)
	if err != nil {
		logrus.Errorf("Failed to update nsm: %s", err)
		return nil, err
	}

	return mapNsmFromCustomResource(nsmCdr), nil
}
